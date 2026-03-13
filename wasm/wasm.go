//go:build js && wasm

package main

import (
	"syscall/js"

	"github.com/bluemir/grim/internal/buildinfo"
	"github.com/bluemir/grim/internal/parser"
	"github.com/bluemir/grim/internal/renderer"
	"github.com/sirupsen/logrus"
)

func main() {
	// 방법 A: 깔끔한 텍스트로만 출력 (색상 끄기)
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})

	// 방법 B: 브라우저 콘솔에서 열어보기 좋은 JSON 형태 (추천)
	// log.SetFormatter(&logrus.JSONFormatter{})

	logrus.SetLevel(logrus.DebugLevel)
	logrus.WithFields(logrus.Fields{
		"app":       buildinfo.AppName,
		"version":   buildinfo.Version,
		"buildTime": buildinfo.BuildTime,
	}).Info("Grim WASM 엔진이 로드되었습니다.")

	quit := make(chan struct{}, 0)

	js.Global().Set("grimRender", js.FuncOf(renderGrim))
	js.Global().Set("grimFormat", js.FuncOf(formatGrim))
	js.Global().Set("grimShutdown", js.FuncOf(func(this js.Value, args []js.Value) any {
		quit <- struct{}{} // 채널에 신호 전송
		return nil
	}))

	// 3. 채널에서 신호가 올 때까지 main 함수 무한 대기 (블로킹)
	<-quit

	// JS에서 shutdownGrim()을 호출하면 대기가 풀리고 프로그램이 정상 종료됨
	logrus.Info("Grim WASM 엔진이 종료되었습니다.")
}

func formatGrim(this js.Value, args []js.Value) any {
	code := args[0].String()
	doc, err := parser.Parse(code)
	if err != nil {
		// Return original code on parse error (safe fallback)
		return js.ValueOf(code)
	}
	return js.ValueOf(parser.Format(doc))
}

func renderGrim(this js.Value, args []js.Value) any {
	code := args[0].String()

	logrus.WithFields(logrus.Fields{
		"action": "parse",
		"length": len(code),
	}).Info("다이어그램 렌더링을 시작합니다.")

	doc, err := parser.Parse(code)
	if err != nil {
		logrus.WithError(err).Warn("파싱 에러 (부분 결과 사용)")
	}

	logrus.WithFields(logrus.Fields{
		"statements": len(doc.Statements),
	}).Debug("파싱 완료")

	svg := renderer.Render(doc)
	return js.ValueOf(svg)
}
