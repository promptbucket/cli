package packager

import (
    "archive/tar"
    "bytes"
    "compress/gzip"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "io"
    "os"

    "gopkg.in/yaml.v3"
)

type Artifact struct {
    Path   string
    Size   int64
    Digest string
}

// Build reads promptbucket.yaml and produces a .pbt package in the current directory.
func Build() (Artifact, error) {
    var art Artifact
    data, err := os.ReadFile(ManifestFile)
    if err != nil {
        return art, err
    }

    var m Manifest
    if err := yaml.Unmarshal(data, &m); err != nil {
        return art, err
    }
    if m.Name == "" || m.Version == "" || m.Licence == "" || m.Prompt == "" {
        return art, fmt.Errorf("manifest missing required fields")
    }

    // tar
    var tarBuf bytes.Buffer
    tw := tar.NewWriter(&tarBuf)
    hdr := &tar.Header{Name: ManifestFile, Mode: 0644, Size: int64(len(data))}
    if err := tw.WriteHeader(hdr); err != nil {
        return art, err
    }
    if _, err := tw.Write(data); err != nil {
        return art, err
    }
    if err := tw.Close(); err != nil {
        return art, err
    }

    // gzip
    var gzBuf bytes.Buffer
    gw := gzip.NewWriter(&gzBuf)
    if _, err := io.Copy(gw, &tarBuf); err != nil {
        return art, err
    }
    if err := gw.Close(); err != nil {
        return art, err
    }

    payload := append([]byte(MagicHeader), gzBuf.Bytes()...)
    sum := sha256.Sum256(payload)
    digest := "sha256:" + hex.EncodeToString(sum[:])

    out := fmt.Sprintf("%s-%s.pbt", m.Name, m.Version)
    if err := os.WriteFile(out, payload, 0644); err != nil {
        return art, err
    }
    info, err := os.Stat(out)
    if err != nil {
        return art, err
    }

    art.Path = out
    art.Size = info.Size()
    art.Digest = digest
    return art, nil
}
