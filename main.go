package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/valyala/fasthttp"
	"log"

  "crypto/tls"
  "net"
  "golang.org/x/crypto/acme"
  "golang.org/x/crypto/acme/autocert"
)

var (
	addr  = flag.String("addr", "0.0.0.0:6060", "TCP address to listen to")
	token = flag.String("token", "", "TheCatAPI token")
)

func main() {
	log.Print("- Loading cat-api-wrapper")

	m := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist("139.162.221.134"), // Replace with your domain.
		Cache:      autocert.DirCache("./certs"),
	}

	cfg := &tls.Config{
		GetCertificate: m.GetCertificate,
		NextProtos: []string{
			"http/1.1", acme.ALPNProto,
		},
	}

	// Let's Encrypt tls-alpn-01 only works on port 443.
	ln, err := net.Listen("tcp4", "0.0.0.0:443") /* #nosec G102 */
	if err != nil {
		panic(err)
	}

	lnTls := tls.NewListener(ln, cfg)

	if err := fasthttp.Serve(lnTls, requestHandler); err != nil {
		panic(err)
	}
}

func requestHandler(ctx *fasthttp.RequestCtx) {
	switch string(ctx.Method()) {
	case fasthttp.MethodGet:
		handleGet(ctx)
	case fasthttp.MethodPost:
		handleApi(ctx)
	default:
		handleError(ctx, 400, "Bad Method")
	}
}

type CatApiObj struct {
	Breeds []string `json:"breeds"`
	ID     string   `json:"id"`
	URL    string   `json:"url"`
	Width  int64    `json:"width"`
	Height int64    `json:"height"`
}

func requestCat(ctx *fasthttp.RequestCtx) CatApiObj {
	body, err := RequestUrl("https://api.thecatapi.com/v1/images/search", "x-api-key", *token)
	if err != nil {
		panic(err)
	}

	log.Printf("body: %s\n", body)
	var objs []CatApiObj
	if err := json.Unmarshal(body, &objs); err != nil {
		panic(err)
	}

	return objs[0]
}

func handleGet(ctx *fasthttp.RequestCtx) {
	switch string(ctx.Path()) {
	case "/api/random":
		cat := requestCat(ctx)
		fmt.Fprintf(ctx, "%s\n", cat.URL)
	default:
		handleError(ctx, 400, "Bad GET path")
	}
}

// handleApi will handle POST requests to the API
func handleApi(ctx *fasthttp.RequestCtx) {
	switch string(ctx.Path()) {
	case "/api/created":
		ctx.SetStatusCode(201)
		fmt.Fprintf(ctx, "201 Created\n")
	default:
		handleError(ctx, 400, "Bad API path")
	}
}

func handleError(ctx *fasthttp.RequestCtx, status int, msg string) {
	ctx.SetStatusCode(status)
	fmt.Fprintf(ctx, "%v %s\n", status, msg)
}

// RequestUrl will return the bytes of the body of url
func RequestUrl(url string, header string, value string) ([]byte, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.SetRequestURI(url)
	req.Header.Set(header, value)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	// Perform the request
	err := fasthttp.Do(req, resp)
	if err != nil {
		fmt.Printf("Client get failed: %s\n", err)
		return nil, err
	}
	return resp.Body(), nil
}
