package domain

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
)

const (
	signAlgorithm = "AWS4-HMAC-SHA256"
)

type Auth struct {
	accessKey string
	secretKey string
}

func NewAuth(accessKey, secretKey string) *Auth {
	return &Auth{accessKey: accessKey, secretKey: secretKey}
}

type authHeader struct {
	credential    string
	signedHeaders string
	signature     string
}

func (a *Auth) parseAuthHeader(header string) (*authHeader, error) {
	prefix := signAlgorithm + " "
	if !strings.HasPrefix(header, prefix) {
		return nil, errors.New("signing algorithm not supported")
	}

	// remove the signing method prefix from the string
	fields := strings.Split(header[len(prefix):], ",")

	kv := make(map[string]string)
	for _, field := range fields {
		parts := strings.SplitN(strings.TrimSpace(field), "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		value := parts[1]
		kv[key] = value
	}

	cred := kv["Credential"]
	prefix = a.accessKey + "/"
	if !strings.HasPrefix(cred, prefix) {
		return nil, errors.New("invalid access key")
	}
	cred = cred[len(prefix):]

	return &authHeader{
		credential:    cred,
		signedHeaders: kv["SignedHeaders"],
		signature:     kv["Signature"],
	}, nil
}

func hmacHash(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}

func (a *Auth) signingKey(cred string) []byte {
	key := []byte("AWS4" + a.secretKey)
	values := strings.Split(cred, "/")
	for _, v := range values {
		key = hmacHash(key, v)
	}
	return key
}

func canonicalUri(uri string) string {
	if len(uri) < 1 {
		return "/"
	}
	return strings.Split(uri, "?")[0]
}

func canonicalQuery(uri string) string {
	// get the query from the uri
	parts := strings.Split(uri, "?")
	query := ""
	if len(parts) > 1 {
		query = parts[1]
	}

	parts = strings.Split(query, "&")
	sort.Strings(parts)

	// rebuild query parameters sorted
	str := ""
	for _, v := range parts {
		str += v + "&"
	}
	if len(str) > 0 {
		str = str[:len(str)-1]
	}

	return str
}

func canonicalHeaders(headers map[string]string, signed []string) string {
	result := ""
	// signed is assumed to be alphabetically sorted
	for _, key := range signed {
		result += fmt.Sprintf("%s:%s\n", key, strings.TrimSpace(headers[key]))
	}
	return result
}

func canonicalRequest(
	method, uri string,
	headers map[string]string,
	signed string,
	body string) string {
	return method + "\n" +
		canonicalUri(uri) + "\n" +
		canonicalQuery(uri) + "\n" +
		canonicalHeaders(headers, strings.Split(signed, ";")) + "\n" +
		signed + "\n" + body
}

func strToSign(algo, date, cred, req string) string {
	return algo + "\n" + date + "\n" + cred + "\n" + Sha256Hash([]byte(req))
}

func (a *Auth) Validate(method, uri string, headers map[string]string, body string) error {
	if len(headers["authorization"]) < 1 {
		return errors.New("authorization header missing")
	}
	authHeader, err := a.parseAuthHeader(headers["authorization"])
	if err != nil {
		return err
	}

	req := canonicalRequest(method, uri, headers, authHeader.signedHeaders, body)
	str := strToSign(signAlgorithm, headers["x-amz-date"], authHeader.credential, req)
	key := a.signingKey(authHeader.credential)

	signature := hex.EncodeToString(hmacHash(key, str))
	if signature != authHeader.signature {
		return errors.New("invalid signature")
	}

	return nil
}
