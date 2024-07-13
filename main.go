package main

import (
	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

// SysDir 系统目录
const (
	logDir      = "log/"
	logFilePath = logDir + "server.log"
)

func init() {
	logInit()
}

func main() {
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-stopCh
		GracefulStop()
		os.Exit(0)
	}()
	if err := Start(); err != nil {
		log.Fatal(err)
	}
}

func logInit() {
	log.SetLevel(log.InfoLevel)
	log.SetReportCaller(true)
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalln("get pwd err: ：", err)
		return
	}
	logWriter := io.MultiWriter(
		os.Stdout,
		&lumberjack.Logger{
			Filename: filepath.Join(cwd, logFilePath),
			MaxSize:  10, // MB
			MaxAge:   2,  //days
		})
	log.SetOutput(logWriter)
	log.Infoln("log 初始化完毕")
}
