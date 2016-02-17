package jwt_test

import (
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/context"

	jwtg "github.com/dgrijalva/jwt-go"
	"github.com/goadesign/goa"
	"github.com/goadesign/middleware/jwt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var signingKey = []byte("jwtsecretsauce")

// Sample data from http://tools.ietf.org/html/draft-jones-json-web-signature-04#appendix-A.1
var hmacTestKey, _ = ioutil.ReadFile("test/hmacTestKey")
var rsaSampleKey, _ = ioutil.ReadFile("test/sample_key")
var rsaSampleKeyPub, _ = ioutil.ReadFile("test/sample_key.pub")

var _ = Describe("JWT Middleware", func() {
	var ctx context.Context
	var spec *jwt.Specification
	var req *http.Request
	var rw http.ResponseWriter
	var token *jwtg.Token
	var tokenString string
	validFunc := func(token *jwtg.Token) (interface{}, error) {
		return signingKey, nil
	}

	BeforeEach(func() {
		var err error
		req, err = http.NewRequest("POST", "/goo", strings.NewReader(`{"payload":42}`))
		Ω(err).ShouldNot(HaveOccurred())
		rw = new(TestResponseWriter)
		s := goa.New("test")
		s.SetEncoder(goa.JSONEncoderFactory(), true, "*/*")
		ctx = goa.NewContext(nil, s, rw, req, nil)
		spec = &jwt.Specification{
			AllowParam:     true,
			ValidationFunc: validFunc,
		}
		token = jwtg.New(jwtg.SigningMethodHS256)
		token.Claims["exp"] = time.Now().Add(time.Hour * 24).Unix()
		token.Claims["random"] = "42"
		tokenString, err = token.SignedString(signingKey)
		Ω(err).ShouldNot(HaveOccurred())
	})

	It("requires a jwt token be present", func() {
		h := func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
			rw.WriteHeader(200)
			rw.Write([]byte("ok"))
			return nil
		}
		jw := jwt.Middleware(spec)(h)
		Ω(jw(ctx, rw, req)).ShouldNot(HaveOccurred())
		Ω(goa.Response(ctx).Status).Should(Equal(http.StatusUnauthorized))

	})

	It("returns the jwt token that was sent as a header", func() {

		req.Header.Set("Authorization", "bearer "+tokenString)
		h := func(c context.Context, rw http.ResponseWriter, req *http.Request) error {
			ctx = c
			return goa.Response(c).Send(ctx, 200, "ok")
		}
		jw := jwt.Middleware(spec)(h)
		Ω(jw(ctx, rw, req)).ShouldNot(HaveOccurred())
		Ω(goa.Response(ctx).Status).Should(Equal(http.StatusOK))
		tok, err := jwtg.Parse(tokenString, validFunc)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(ctx.Value(jwt.JWTKey)).Should(Equal(tok))
	})

	It("returns the custom claims", func() {
		req.Header.Set("Authorization", "bearer "+tokenString)
		h := func(c context.Context, rw http.ResponseWriter, req *http.Request) error {
			ctx = c
			return goa.Response(c).Send(ctx, 200, "ok")
		}
		jw := jwt.Middleware(spec)(h)
		Ω(jw(ctx, rw, req)).ShouldNot(HaveOccurred())
		Ω(goa.Response(ctx).Status).Should(Equal(http.StatusOK))
		tok, err := jwtg.Parse(tokenString, validFunc)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(ctx.Value(jwt.JWTKey)).Should(Equal(tok))
		ctxtok := ctx.Value(jwt.JWTKey).(*jwtg.Token)
		clms := ctxtok.Claims
		Ω(clms["random"]).Should(Equal("42"))
	})

	It("returns the jwt token that was sent as a querystring", func() {
		var err error
		req, err = http.NewRequest("POST", "/goo?token="+tokenString, strings.NewReader(`{"payload":42}`))
		Ω(err).ShouldNot(HaveOccurred())
		spec = &jwt.Specification{
			AllowParam:     true,
			ValidationFunc: validFunc,
		}
		h := func(c context.Context, rw http.ResponseWriter, req *http.Request) error {
			ctx = c
			return goa.Response(c).Send(ctx, 200, "ok")
		}
		jw := jwt.Middleware(spec)(h)
		Ω(jw(ctx, rw, req)).ShouldNot(HaveOccurred())
		Ω(goa.Response(ctx).Status).Should(Equal(http.StatusOK))
		tok, err := jwtg.Parse(tokenString, validFunc)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(ctx.Value(jwt.JWTKey)).Should(Equal(tok))
	})

})
var _ = Describe("JWT Token HMAC", func() {
	var claims map[string]interface{}
	var spec *jwt.Specification
	var tm *jwt.TokenManager
	validFunc := func() (interface{}, error) {
		return hmacTestKey, nil
	}
	keyFunc := func(*jwtg.Token) (interface{}, error) {
		return hmacTestKey, nil
	}
	spec = &jwt.Specification{
		Issuer:           "goa",
		TTLMinutes:       20,
		KeySigningMethod: jwt.HMAC256,
		SigningKeyFunc:   validFunc,
	}
	tm = jwt.NewTokenManager(spec)
	BeforeEach(func() {
		claims = make(map[string]interface{})

		claims["randomstring"] = "43"

	})

	It("creates a valid token", func() {
		tok, err := tm.Create(claims)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(len(tok)).ShouldNot(BeZero())
	})
	It("contains the intended claims", func() {
		tok, err := tm.Create(claims)
		Ω(err).ShouldNot(HaveOccurred())
		rettok, err := jwtg.Parse(tok, keyFunc)
		Ω(err).ShouldNot(HaveOccurred())
		rndmstring := rettok.Claims["randomstring"].(string)
		Ω(rndmstring).Should(Equal("43"))
	})

})
var _ = Describe("JWT Token RSA", func() {
	var claims map[string]interface{}
	var spec *jwt.Specification
	var tm *jwt.TokenManager
	validFunc := func() (interface{}, error) {
		return rsaSampleKey, nil
	}
	keyFunc := func(*jwtg.Token) (interface{}, error) {
		return rsaSampleKeyPub, nil
	}
	spec = &jwt.Specification{
		Issuer:           "goa",
		TTLMinutes:       20,
		KeySigningMethod: jwt.RSA256,
		SigningKeyFunc:   validFunc,
	}
	tm = jwt.NewTokenManager(spec)
	BeforeEach(func() {
		claims = make(map[string]interface{})

		claims["randomstring"] = "43"

	})

	It("creates a valid token", func() {
		tok, err := tm.Create(claims)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(len(tok)).ShouldNot(BeZero())
	})
	It("contains the intended claims", func() {
		tok, err := tm.Create(claims)
		Ω(err).ShouldNot(HaveOccurred())
		rettok, err := jwtg.Parse(tok, keyFunc)
		Ω(err).ShouldNot(HaveOccurred())
		rndmstring := rettok.Claims["randomstring"].(string)
		Ω(rndmstring).Should(Equal("43"))
	})

})

type TestResponseWriter struct {
	ParentHeader http.Header
	Body         []byte
	Status       int
}

func (t *TestResponseWriter) Header() http.Header {
	return t.ParentHeader
}

func (t *TestResponseWriter) Write(b []byte) (int, error) {
	t.Body = append(t.Body, b...)
	return len(b), nil
}

func (t *TestResponseWriter) WriteHeader(s int) {
	t.Status = s
}
