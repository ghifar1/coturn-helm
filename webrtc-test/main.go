package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pion/webrtc/v3"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	turnDomain := envDefault("TURN_DOMAIN", "turn.server.ghifari.dev")
	if turnDomain == "" {
		log.Fatal("TURN_DOMAIN must not be empty")
	}

	turnPort := envInt("TURN_PORT", 3478)
	turnTLSPort := envInt("TURN_TLS_PORT", 5349)
	turnUser := strings.TrimSpace(os.Getenv("TURN_USERNAME"))
	turnPass := strings.TrimSpace(os.Getenv("TURN_PASSWORD"))

	if authSecret := strings.TrimSpace(os.Getenv("TURN_AUTH_SECRET")); authSecret != "" {
		expires := time.Now().Add(30 * time.Minute).Unix()
		generatedUser := fmt.Sprintf("%d:%s", expires, "webrtc-turn-check")
		mac := hmac.New(sha1.New, []byte(authSecret))
		_, _ = mac.Write([]byte(generatedUser))
		generatedPass := base64.StdEncoding.EncodeToString(mac.Sum(nil))
		log.Printf("generated TURN REST credentials with expiry %d", expires)
		turnUser = generatedUser
		turnPass = generatedPass
	}

	iceServers := make([]webrtc.ICEServer, 0, 2)

	udpURLs := []string{
		fmt.Sprintf("turn:%s:%d?transport=udp", turnDomain, turnPort),
		fmt.Sprintf("turn:%s:%d?transport=tcp", turnDomain, turnPort),
	}
	iceServers = append(iceServers, webrtc.ICEServer{
		URLs:           udpURLs,
		Username:       turnUser,
		Credential:     turnPass,
		CredentialType: webrtc.ICECredentialTypePassword,
	})

	if turnTLSPort > 0 {
		iceServers = append(iceServers, webrtc.ICEServer{
			URLs:           []string{fmt.Sprintf("turns:%s:%d?transport=tcp", turnDomain, turnTLSPort)},
			Username:       turnUser,
			Credential:     turnPass,
			CredentialType: webrtc.ICECredentialTypePassword,
		})
	}

	config := webrtc.Configuration{
		ICEServers:         iceServers,
		ICETransportPolicy: webrtc.ICETransportPolicyRelay,
	}

	api := webrtc.NewAPI()

	offerPeer, err := api.NewPeerConnection(config)
	if err != nil {
		log.Fatalf("create offer peer: %v", err)
	}
	defer func() {
		_ = offerPeer.Close()
	}()

	answerPeer, err := api.NewPeerConnection(config)
	if err != nil {
		log.Fatalf("create answer peer: %v", err)
	}
	defer func() {
		_ = answerPeer.Close()
	}()

	var once sync.Once
	errCh := make(chan error, 4)
	doneCh := make(chan struct{}, 1)
	messagesNeeded := 2
	var msgLock sync.Mutex

	markMessage := func(source string, payload string) {
		msgLock.Lock()
		defer msgLock.Unlock()
		messagesNeeded--
		log.Printf("%s received message: %s (remaining confirmations: %d)", source, payload, messagesNeeded)
		if messagesNeeded <= 0 {
			once.Do(func() {
				doneCh <- struct{}{}
			})
		}
	}

	offerPeer.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("offer peer ICE state: %s", state.String())
		switch state {
		case webrtc.ICEConnectionStateFailed, webrtc.ICEConnectionStateDisconnected:
			errCh <- fmt.Errorf("offer peer ICE state: %s", state.String())
		}
	})

	answerPeer.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("answer peer ICE state: %s", state.String())
		switch state {
		case webrtc.ICEConnectionStateFailed, webrtc.ICEConnectionStateDisconnected:
			errCh <- fmt.Errorf("answer peer ICE state: %s", state.String())
		}
	})

	offerPeer.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		log.Printf("offer peer gathered candidate: %s (%s)", c.Address, c.Typ)
		if err := answerPeer.AddICECandidate(c.ToJSON()); err != nil {
			errCh <- fmt.Errorf("answer peer add candidate: %w", err)
		}
	})

	answerPeer.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		log.Printf("answer peer gathered candidate: %s (%s)", c.Address, c.Typ)
		if err := offerPeer.AddICECandidate(c.ToJSON()); err != nil {
			errCh <- fmt.Errorf("offer peer add candidate: %w", err)
		}
	})

	offerChannel, err := offerPeer.CreateDataChannel("demo", nil)
	if err != nil {
		log.Fatalf("create data channel: %v", err)
	}

	offerChannel.OnOpen(func() {
		log.Printf("offer data channel open, sending greeting")
		if err := offerChannel.SendText("hello-from-offer"); err != nil {
			errCh <- fmt.Errorf("offer channel send: %w", err)
		}
	})

	offerChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		markMessage("offer", string(msg.Data))
	})

	answerPeer.OnDataChannel(func(dc *webrtc.DataChannel) {
		log.Printf("answer data channel created: %s", dc.Label())

		dc.OnOpen(func() {
			log.Printf("answer data channel open, sending reply")
			if err := dc.SendText("hello-from-answer"); err != nil {
				errCh <- fmt.Errorf("answer channel send: %w", err)
			}
		})

		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			markMessage("answer", string(msg.Data))
		})
	})

	offer, err := offerPeer.CreateOffer(nil)
	if err != nil {
		log.Fatalf("create offer: %v", err)
	}
	if err := offerPeer.SetLocalDescription(offer); err != nil {
		log.Fatalf("set local description (offer): %v", err)
	}

	if err := answerPeer.SetRemoteDescription(*offerPeer.LocalDescription()); err != nil {
		log.Fatalf("set remote description (answer side): %v", err)
	}

	answer, err := answerPeer.CreateAnswer(nil)
	if err != nil {
		log.Fatalf("create answer: %v", err)
	}
	if err := answerPeer.SetLocalDescription(answer); err != nil {
		log.Fatalf("set local description (answer): %v", err)
	}

	if err := offerPeer.SetRemoteDescription(*answerPeer.LocalDescription()); err != nil {
		log.Fatalf("set remote description (offer side): %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	select {
	case <-doneCh:
		log.Printf("WebRTC relay over TURN succeeded via %s", turnDomain)
	case err := <-errCh:
		log.Fatalf("WebRTC exchange failed: %v", err)
	case <-ctx.Done():
		log.Fatalf("timeout waiting for WebRTC completion")
	}
}

func envDefault(key, def string) string {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		return val
	}
	return def
}

func envInt(key string, def int) int {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return def
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		log.Printf("invalid value for %s=%q, using default %d", key, val, def)
		return def
	}
	return parsed
}
