package main

import (
	"log"

	asr "gosdk/client"
)

type RecognitionListener struct{
	doneChan chan struct{}
}

func (l *RecognitionListener) OnRecognitionStart(resp *asr.RecognitionResponse) {
	log.Printf("[ 🚀 Event ] Recognition started ID-%s", resp.VoiceID)
}

func (l *RecognitionListener) OnSentenceBegin(resp *asr.RecognitionResponse) {
	log.Printf("[ 🚀 Event ] Sentence begin ID-%s", resp.VoiceID)
}

func (l *RecognitionListener) OnRecognitionResultChange(resp *asr.RecognitionResponse) {
	log.Printf("[ 🚀 Event ] Partial result: %s\n", resp.Result.Text)
}

func (l *RecognitionListener) OnSentenceEnd(resp *asr.RecognitionResponse) {
	log.Printf("[ 🚀 Event ] Sentence end: %s", resp.Result.Text)
}

func (l *RecognitionListener) OnRecognitionComplete(resp *asr.RecognitionResponse) {
	log.Printf("[ 🚀 Event ] Recognition complete ID-%s ==> Text:%s", resp.VoiceID,resp.Result.Text)
	if l.doneChan != nil {
		close(l.doneChan)
		l.doneChan = nil
	}
}

func (l *RecognitionListener) OnFail(resp *asr.RecognitionResponse, err error) {
	log.Printf("[ Event ] Recognition failed: %v", err)
}