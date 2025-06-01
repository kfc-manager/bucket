package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode"
)

type Storage struct {
	path string
}

func NewStorage(path string) (*Storage, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("path '%s' does not exist", path)
	} else if err != nil {
		return nil, err
	} else if !info.IsDir() {
		return nil, fmt.Errorf("path '%s' is not a directory", path)
	}
	return &Storage{path: path}, nil
}

// implemented naming rules from the following link:
// https://docs.aws.amazon.com/AmazonS3/latest/userguide/bucketnamingrules.html
func validName(name string) error {
	// Bucket names must be between 3 (min) and 63 (max) characters long.
	if len(name) < 3 {
		return &Error{
			msg:    "bucket name must be at least 3 characters long",
			Status: http.StatusBadRequest,
		}
	}
	if len(name) > 63 {
		return &Error{
			msg:    "bucket name must be not longer than 63 characters",
			Status: http.StatusBadRequest,
		}
	}

	// Bucket names can consist only of lowercase letters, numbers, periods (.), and hyphens (-).
	for _, char := range name {
		if unicode.IsLower(char) || unicode.IsDigit(char) || string(char) == "." || string(char) == "-" {
			continue
		}
		return &Error{
			msg:    "bucket name can consist only of lowercase letters, numbers, periods and hyphens",
			Status: http.StatusBadRequest,
		}
	}

	// Bucket names must begin and end with a letter or number.
	if begin := rune(name[:1][0]); !unicode.IsLower(begin) && !unicode.IsDigit(begin) {
		return &Error{
			msg:    "bucket name must begin with a letter or number",
			Status: http.StatusBadRequest,
		}
	}
	if end := rune(name[len(name)-1:][0]); !unicode.IsLower(end) && !unicode.IsDigit(end) {
		return &Error{
			msg:    "bucket name must end with a letter or number",
			Status: http.StatusBadRequest,
		}
	}

	// Bucket names must not contain two adjacent periods.
	if strings.Contains(name, "..") {
		return &Error{
			msg:    "bucket name can not contain two adjacent periods",
			Status: http.StatusBadRequest,
		}
	}

	// Bucket names must not be formatted as an IP address (for example, 192.168.5.4).
	if net.ParseIP(name) != nil {
		return &Error{
			msg:    "bucket name can not be formatted as an IP address",
			Status: http.StatusBadRequest,
		}
	}

	// Bucket names must not start with the prefix xn--.
	// Bucket names must not start with the prefix sthree-.
	// Bucket names must not start with the prefix amzn-s3-demo-
	prefix := []string{"xn--", "sthree-", "amzn-s3-demo-"}
	for _, p := range prefix {
		if strings.HasPrefix(name, p) {
			return &Error{
				msg:    fmt.Sprintf("bucket name can not begin with the prefix: '%s'", p),
				Status: http.StatusBadRequest,
			}
		}
	}

	// Bucket names must not end with the suffix -s3alias. This suffix is reserved
	// for access point alias names. For more information, see Access point for general
	// purpose buckets aliases.
	// Bucket names must not end with the suffix --ol-s3. This suffix is reserved for
	// Object Lambda Access Point alias names. For more information, see How to use a
	// bucket-style alias for your S3 bucket Object Lambda Access Point.
	// Bucket names must not end with the suffix .mrap. This suffix is reserved for
	// Multi-Region Access Point names. For more information, see Rules for naming
	// Amazon S3 Multi-Region Access Points.
	// Bucket names must not end with the suffix --x-s3. This suffix is reserved for
	// directory buckets. For more information, see Directory bucket naming rules.
	// Bucket names must not end with the suffix --table-s3. This suffix is reserved for
	// S3 Tables buckets. For more information, see Amazon S3 table bucket, table, and
	// namespace naming rules.
	suffix := []string{"-s3alias", "--ol-s3", "--x-s3", "--table-s3"}
	for _, s := range suffix {
		if strings.HasSuffix(name, s) {
			return &Error{
				msg:    fmt.Sprintf("bucket name can not end with the suffix: '%s'", s),
				Status: http.StatusBadRequest,
			}
		}
	}

	return nil
}

func (s *Storage) existPath(path string) bool {
	_, err := os.Stat(s.path + "/" + path)
	return err == nil
}

func (s *Storage) NewBucket(name string) error {
	if err := validName(name); err != nil {
		return err
	}
	if s.existPath(name) {
		return &Error{
			msg:    "requested bucket name is not available",
			Status: http.StatusConflict,
		}
	}

	if err := os.MkdirAll(s.path+"/"+name, 0755); err != nil {
		return err
	}

	return nil
}

type metadata struct {
	ContentHash  string `json:"content_sha256"`
	ContentSize  int    `json:"content_size"`
	OriginalKey  string `json:"original_key"`
	LastModified int64  `json:"last_modified"`
}

func (s *Storage) Get(bucket, key string) ([]byte, error) {
	if !s.existPath(bucket) {
		return nil, &Error{
			msg:    "requested bucket does not exist",
			Status: http.StatusNotFound,
		}
	}

	hash := Sha256Hash([]byte(key))
	if !s.existPath(bucket + "/" + hash) {
		return nil, &Error{
			msg:    "object under requested key does not exist",
			Status: http.StatusNotFound,
		}
	}

	path := s.path + "/" + bucket + "/" + hash
	body, err := os.ReadFile(path + "/body")
	if err != nil {
		return nil, fmt.Errorf("could not read data file: %w", err)
	}

	meta := &metadata{}
	b, err := os.ReadFile(path + "/metadata.json")
	if err != nil {
		return nil, fmt.Errorf("could not read metadata file: %w", err)
	}
	if err := json.Unmarshal(b, meta); err != nil {
		return nil, fmt.Errorf("could not unmarshal metadata.json content: %w", err)
	}

	if Sha256Hash(body) != meta.ContentHash {
		return nil, errors.New("content checksum mismatch")
	}

	return body, nil
}

func (s *Storage) Put(bucket, key string, body []byte) error {
	if !s.existPath(bucket) {
		return &Error{
			msg:    "requested bucket does not exist",
			Status: http.StatusNotFound,
		}
	}

	hash := Sha256Hash([]byte(key))
	// create directory namespace so we can store
	// metadata next to the file content
	dir := s.path + "/" + bucket + "/" + hash
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	meta, err := json.Marshal(&metadata{
		ContentHash:  Sha256Hash(body),
		ContentSize:  len(body),
		OriginalKey:  key,
		LastModified: time.Now().UTC().Unix(),
	})
	if err != nil {
		return fmt.Errorf("could not marshal metadata struct: %w", err)
	}
	if err := os.WriteFile(dir+"/metadata.json", meta, 0644); err != nil {
		return fmt.Errorf("could not write metadata.json: %w", err)
	}

	return os.WriteFile(dir+"/body", body, 0644)
}

func (s *Storage) Delete(bucket, key string) error {
	if !s.existPath(bucket) {
		return &Error{
			msg:    "requested bucket does not exist",
			Status: http.StatusNotFound,
		}
	}

	hash := Sha256Hash([]byte(key))
	if !s.existPath(bucket + "/" + hash) {
		return &Error{
			msg:    "object under requested key does not exist",
			Status: http.StatusNotFound,
		}
	}

	return os.RemoveAll(s.path + "/" + bucket + "/" + hash)
}
