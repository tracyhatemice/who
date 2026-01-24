package ddns

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const route53Host = "route53.amazonaws.com"

// XML structures for Route53 API
type changeRequest struct {
	XMLName     xml.Name    `xml:"ChangeResourceRecordSetsRequest"`
	XMLNS       string      `xml:"xmlns,attr"`
	ChangeBatch changeBatch `xml:"ChangeBatch"`
}

type changeBatch struct {
	Changes []change `xml:"Changes>Change"`
}

type change struct {
	Action            string            `xml:"Action"`
	ResourceRecordSet resourceRecordSet `xml:"ResourceRecordSet"`
}

type resourceRecordSet struct {
	Name            string           `xml:"Name"`
	Type            string           `xml:"Type"`
	TTL             int              `xml:"TTL"`
	ResourceRecords []resourceRecord `xml:"ResourceRecords>ResourceRecord"`
}

type resourceRecord struct {
	Value string `xml:"Value"`
}

// Route53 implements the Provider interface for AWS Route53.
type Route53 struct {
	accessKey string
	secretKey string
	zoneID    string
	client    *http.Client
}

// NewRoute53 creates a new Route53 provider.
func NewRoute53(accessKey, secretKey, zoneID string) *Route53 {
	return &Route53{
		accessKey: accessKey,
		secretKey: secretKey,
		zoneID:    zoneID,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

// Update implements Provider.Update for Route53.
func (r *Route53) Update(domain, ip string, ttl int) error {
	// Determine record type based on IP format
	recordType := "A"
	if strings.Contains(ip, ":") {
		recordType = "AAAA"
	}

	// Build the XML payload for UPSERT
	body, err := r.buildChangeXML(domain, ip, recordType, ttl)
	if err != nil {
		return fmt.Errorf("building XML: %w", err)
	}

	// Build the request
	url := fmt.Sprintf("https://%s/2013-04-01/hostedzone/%s/rrset", route53Host, r.zoneID)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	// Sign the request with AWS v4 signature
	r.signRequest(req, body)

	// Execute request
	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("route53 returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (r *Route53) buildChangeXML(domain, ip, recordType string, ttl int) ([]byte, error) {
	// Ensure domain ends with a dot (FQDN)
	if !strings.HasSuffix(domain, ".") {
		domain = domain + "."
	}

	req := changeRequest{
		XMLNS: "https://route53.amazonaws.com/doc/2013-04-01/",
		ChangeBatch: changeBatch{
			Changes: []change{{
				Action: "UPSERT",
				ResourceRecordSet: resourceRecordSet{
					Name: domain,
					Type: recordType,
					TTL:  ttl,
					ResourceRecords: []resourceRecord{{
						Value: ip,
					}},
				},
			}},
		},
	}

	return xml.Marshal(req)
}

// signRequest adds AWS v4 signature headers to the request.
func (r *Route53) signRequest(req *http.Request, payload []byte) {
	now := time.Now().UTC()
	dateStamp := now.Format("20060102")
	amzDate := now.Format("20060102T150405Z")

	// Required headers
	req.Header.Set("Content-Type", "text/xml")
	req.Header.Set("Host", route53Host)
	req.Header.Set("X-Amz-Date", amzDate)

	// AWS v4 signature calculation
	service := "route53"
	region := "us-east-1" // Route53 always uses us-east-1
	algorithm := "AWS4-HMAC-SHA256"

	// Create canonical request
	payloadHash := sha256Hash(payload)
	canonicalHeaders := fmt.Sprintf("content-type:%s\nhost:%s\nx-amz-date:%s\n",
		req.Header.Get("Content-Type"), req.Header.Get("Host"), amzDate)
	signedHeaders := "content-type;host;x-amz-date"

	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		req.Method,
		req.URL.Path,
		"", // query string (empty)
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	)

	// Create string to sign
	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", dateStamp, region, service)
	stringToSign := fmt.Sprintf("%s\n%s\n%s\n%s",
		algorithm,
		amzDate,
		credentialScope,
		sha256Hash([]byte(canonicalRequest)),
	)

	// Calculate signature
	signingKey := r.getSignatureKey(dateStamp, region, service)
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	// Add authorization header
	authHeader := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm, r.accessKey, credentialScope, signedHeaders, signature)
	req.Header.Set("Authorization", authHeader)
}

func (r *Route53) getSignatureKey(dateStamp, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+r.secretKey), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	return kSigning
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func sha256Hash(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
