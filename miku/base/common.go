package base

import (
	"archive/zip"
	"bytes"
	"fmt"
	"github.com/hashicorp/go-hclog"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"path/filepath"
)

const confFilePath = "mikuConf.yaml"

var MkConfig *Config

func init() {
	err := configInit()
	if err != nil {
		log.Error(fmt.Sprintf("MkConfig init error: %s", err))
		os.Exit(1)
	}
}

type MikuLog struct {
	hclog.Logger
}

func BuildLogger(name string, l hclog.Level, output io.Writer) hclog.Logger {
	return hclog.New(&hclog.LoggerOptions{
		Name:            name,
		Level:           l,
		Output:          output,
		TimeFormat:      "2006-01-02 15:04:05",
		Color:           hclog.AutoColor,
		ColorHeaderOnly: true,
	})
}

type Config struct {
	RaftRetainSnapshotCount int    `yaml:"raft_retain_snapshot_count"`
	JoinHandleAddr          string `yaml:"join_handle_addr"`
	RaftNodeAddr            string `yaml:"raft_node_addr"`
	RaftNodeID              string `yaml:"raft_node_id"`
	JoinAddr                string `yaml:"join_addr"`
}

func configInit() error {
	cf, err := os.ReadFile(confFilePath)
	if err != nil {
		return fmt.Errorf("read MkConfig file error: %s", err)
	}

	MkConfig = &Config{}
	if err := yaml.Unmarshal(cf, &MkConfig); err != nil {
		return fmt.Errorf("unmarshal MkConfig file error: %s", err)
	}
	log.Info("MkConfig 初始化完毕", "MkConfig", MkConfig)
	return nil
}

func ZipBackup(source, target string) error {
	zipFile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	err = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return zipProcess(path, info, source, zipWriter)
	})

	return err
}

func zipProcess(path string, info os.FileInfo, source string, zipWriter *zip.Writer) error {
	relPath, err := filepath.Rel(source, path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		_, err = zipWriter.Create(relPath + "/")
		return err
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	zipFileWriter, err := zipWriter.Create(relPath)
	if err != nil {
		return err
	}

	_, err = io.Copy(zipFileWriter, file)
	return err
}

func UnzipBackup(rc io.ReadCloser, target string) error {
	data, err := io.ReadAll(rc)
	if err != nil {
		return err
	}

	reader := bytes.NewReader(data)
	zipReader, err := zip.NewReader(reader, int64(len(data)))
	if err != nil {
		return err
	}

	for _, f := range zipReader.File {
		if err := unzipProcess(f, target); err != nil {
			return err
		}
	}
	return nil
}

func unzipProcess(f *zip.File, target string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	path := filepath.Join(target, f.Name)
	if f.FileInfo().IsDir() {
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			return err
		}
	} else {
		file, err := os.Create(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(file, rc)
		if err != nil {
			return err
		}
	}
	return nil
}
