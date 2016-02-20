package cors

import (
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/net/context"

	"github.com/goadesign/goa"
)

const (
	acAllowCredentials = "Access-Control-Allow-Credentials"
	acAllowHeaders     = "Access-Control-Allow-Headers"
	acAllowMethods     = "Access-Control-Allow-Methods"
	acAllowOrigin      = "Access-Control-Allow-Origin"
	acExposeHeaders    = "Access-Control-Expose-Headers"
	acMaxAge           = "Access-Control-Max-Age"
	acRequestMethod    = "Access-Control-Request-Method"
	acRequestHeaders   = "Access-Control-Request-Headers"
)

// Middleware returns a goa middleware which implements the given CORS specification.
func Middleware(spec Specification) goa.Middleware {
	return func(h goa.Handler) goa.Handler {
		return func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
			header := req.Header
			origin := header.Get("Origin")
			if origin == "" {
				origin = header.Get("X-Origin")
			}
			var res *ResourceDefinition
			var originHeader string
			if origin != "" {
				originHeader = origin
				res = spec.RequestResource(ctx, origin)
				if res == nil {
					goto handleRequest
				}
				acMethod := strings.ToUpper(header.Get(acRequestMethod))
				if req.Method != "OPTIONS" || acMethod == "" {
					goto handleRequest
				}
				found := false
				for _, m := range res.Methods {
					if m == acMethod {
						found = true
						break
					}
				}
				if !found {
					goto handleRequest
				}
				// We are responding to a preflight request.
				headers := header[acRequestHeaders]
				var splat []string
				for _, h := range headers {
					hs := strings.Split(h, ",")
					for _, s := range hs {
						splat = append(splat, strings.TrimSpace(s))
					}
				}
				headers = splat
				if len(headers) > 0 {
					ok := false
					for _, h := range headers {
						for _, h2 := range res.Headers {
							if h2 == "*" || h == h2 {
								ok = true
								break
							}
						}
						if !ok {
							break
						}
					}
					if !ok {
						goto handleRequest
					}
				}
				if res.Origin == "*" && !res.Credentials {
					originHeader = "*"
				}
				rw.Header().Set(acAllowHeaders, strings.Join(headers, ", "))
			}
		handleRequest:
			if res != nil {
				// Apply CORS headers if CORS request
				res.FillHeaders(originHeader, rw.Header())
			} else {
				res = spec.PathResource(req.URL.Path)
			}
			if res != nil {
				// Now apply Vary header (always)
				v := header["Vary"]
				if len(res.Vary) > 0 {
					v = append(v, res.Vary...)
				} else {
					v = append(v, "Origin")
				}
				header["Vary"] = v
			}
			return h(ctx, rw, req)
		}
	}
}

// MountPreflightController mounts the handlers for the CORS preflight requests onto service.
func MountPreflightController(service goa.Service, spec Specification) {
	for _, res := range spec {
		path := res.Path
		if res.IsPathPrefix {
			if strings.HasSuffix(path, "/") {
				path += "*cors"
			} else {
				path += "/*cors"
			}
		}
		handle := service.ServeMux().Lookup("OPTIONS", path)
		if handle == nil {
			h := func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
				// The middleware did all the work of checking already
				rw.Header().Set("Content-Type", "text/plain")
				rw.WriteHeader(200)
				return nil
			}
			wrapped := Middleware(spec)(h)
			ctrl := service.NewController("cors")
			service.ServeMux().Handle("OPTIONS", path, ctrl.HandleFunc("preflight", wrapped, nil))
		}
	}
}

// FillHeaders initializes the given header with the resource CORS headers. origin is the request
// origin.
func (res *ResourceDefinition) FillHeaders(origin string, dest http.Header) {
	dest.Set(acAllowOrigin, origin)
	dest.Set(acAllowMethods, strings.Join(res.Methods, ", "))
	if len(res.Expose) > 0 {
		dest.Set(acExposeHeaders, strings.Join(res.Expose, ", "))
	}
	if res.MaxAge > 0 {
		dest.Set(acMaxAge, strconv.Itoa(res.MaxAge))
	}
	if res.Credentials {
		dest.Set(acAllowCredentials, "true")
	}
}

func (res *ResourceDefinition) OriginAllowed(origin string) bool {
	if res.Origin != "" {
		return res.Origin == "*" || res.Origin == origin
	}
	return res.OriginRegexp.MatchString(origin)
}

// PathMatches returns true if the resource lives under the given path.
func (res *ResourceDefinition) PathMatches(path string) bool {
	if res.IsPathPrefix {
		return strings.HasPrefix(path, res.Path)
	}
	return path == res.Path
}

// RequestResource returns the resource targeted by the CORS request defined in ctx.
func (v Specification) RequestResource(ctx context.Context, origin string) *ResourceDefinition {
	path := goa.Request(ctx).URL.Path
	var match *ResourceDefinition
	for _, res := range v {
		if res.OriginAllowed(origin) && res.PathMatches(path) {
			if res.Check == nil || res.Check(ctx) {
				match = res
				break
			}
		}
	}
	return match
}

// PathResource returns the resource under the given path if any.
func (v Specification) PathResource(path string) *ResourceDefinition {
	var res *ResourceDefinition
	for _, r := range v {
		if r.IsPathPrefix {
			if strings.HasPrefix(path, r.Path) {
				res = r
				break
			}
		} else if r.Path == path {
			res = r
			break
		}
	}
	return res
}
