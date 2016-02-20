package cors_test

import (
	"fmt"
	"net/http"
	"time"

	"golang.org/x/net/context"

	"github.com/goadesign/goa"
	"github.com/goadesign/middleware/cors"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Middleware", func() {

	Context("with a running goa app", func() {
		var dsl func()
		var method string
		var path string
		var optionsHandler goa.Handler

		var service *goa.GracefulApplication
		var url string
		portIndex := 1

		JustBeforeEach(func() {
			goa.Log = nil
			service = goa.NewGraceful("", false).(*goa.GracefulApplication)
			spec, err := cors.New(dsl)
			Ω(err).ShouldNot(HaveOccurred())
			service.Use(cors.Middleware(spec))
			h := func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
				rw.WriteHeader(200)
				return nil
			}
			ctrl := service.NewController("test")
			service.ServeMux().Handle(method, path, ctrl.HandleFunc("", h, nil))
			if optionsHandler != nil {
				service.ServeMux().Handle("OPTIONS", path, ctrl.HandleFunc("", optionsHandler, nil))
			}
			cors.MountPreflightController(service, spec)
			portIndex++
			port := 54511 + portIndex
			url = fmt.Sprintf("http://localhost:%d", port)
			go service.ListenAndServe(fmt.Sprintf(":%d", port))
			// ugh - does anyone have a better idea? we need to wait for the server
			// to start listening or risk tests failing because sendind requests too
			// early.
			time.Sleep(time.Duration(100) * time.Millisecond)
		})

		AfterEach(func() {
			service.Shutdown()
		})

		Context("handling GET requests", func() {
			BeforeEach(func() {
				method = "GET"
				path = "/"
			})

			It("responds", func() {
				resp, err := http.Get(url)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(resp.StatusCode).Should(Equal(200))
			})

			Context("using CORS that allows the request", func() {
				BeforeEach(func() {
					dsl = func() {
						cors.Origin("http://authorized.com", func() {
							cors.Resource("/", func() {
								cors.Methods("GET")
							})
						})
					}
				})

				It("sets the Acess-Control-Allow-Methods header", func() {
					req, err := http.NewRequest("GET", url, nil)
					Ω(err).ShouldNot(HaveOccurred())
					req.Header.Set("Origin", "http://authorized.com")
					resp, err := http.DefaultClient.Do(req)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(resp.StatusCode).Should(Equal(200))
					Ω(resp.Header).Should(HaveKey("Access-Control-Allow-Methods"))
				})

			})

			Context("using CORS that disallows the request", func() {
				BeforeEach(func() {
					dsl = func() {
						cors.Origin("http://authorized.com", func() {
							cors.Resource("/", func() {
								cors.Methods("POST")
							})
						})
					}
				})

				It("does not set the Acess-Control-Allow-Methods header", func() {
					req, err := http.NewRequest("GET", url, nil)
					Ω(err).ShouldNot(HaveOccurred())
					req.Header.Set("Origin", "http://nonauthorized.com")
					resp, err := http.DefaultClient.Do(req)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(resp.StatusCode).Should(Equal(200))
					Ω(resp.Header).ShouldNot(HaveKey("Access-Control-Allow-Methods"))
				})

			})

			Context("using a CORS preflight request", func() {
				BeforeEach(func() {
					dsl = func() {
						cors.Origin("http://authorized.com", func() {
							cors.Resource("/", func() {
								cors.Methods("GET")
							})
						})
					}
				})

				It("sets the Acess-Control-Allow-Methods header when no OPTION action exists", func() {
					req, err := http.NewRequest("OPTIONS", url, nil)
					Ω(err).ShouldNot(HaveOccurred())
					req.Header.Set("Origin", "http://authorized.com")
					req.Header.Set("Access-Control-Request-Method", "GET")
					resp, err := http.DefaultClient.Do(req)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(resp.StatusCode).Should(Equal(200))
					Ω(resp.Header).Should(HaveKey("Access-Control-Allow-Methods"))
				})

				Context("with an OPTIONS action", func() {
					BeforeEach(func() {
						optionsHandler = func(ctx context.Context, rw http.ResponseWriter, req *http.Request) error {
							goa.Response(ctx).WriteHeader(200)
							return nil
						}
					})

					It("sets the Acess-Control-Allow-Methods header when OPTION actions exist", func() {
						req, err := http.NewRequest("OPTIONS", url, nil)
						Ω(err).ShouldNot(HaveOccurred())
						req.Header.Set("Origin", "http://authorized.com")
						req.Header.Set("Access-Control-Request-Method", "GET")
						resp, err := http.DefaultClient.Do(req)
						Ω(err).ShouldNot(HaveOccurred())
						Ω(resp.StatusCode).Should(Equal(200))
						Ω(resp.Header).Should(HaveKey("Access-Control-Allow-Methods"))
					})

				})
			})

			Context("using a CORS preflight request with header access request", func() {
				const header = "foo"

				BeforeEach(func() {
					dsl = func() {
						cors.Origin("http://authorized.com", func() {
							cors.Resource("/", func() {
								cors.Methods("GET")
								cors.Headers(header)
							})
						})
					}
				})

				It("sets the Acess-Control-Allow-Headers header when OPTION actions exist", func() {
					req, err := http.NewRequest("OPTIONS", url, nil)
					Ω(err).ShouldNot(HaveOccurred())
					req.Header.Set("Origin", "http://authorized.com")
					req.Header.Set("Access-Control-Request-Method", "GET")
					req.Header.Set("Access-Control-Request-Headers", header)
					resp, err := http.DefaultClient.Do(req)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(resp.StatusCode).Should(Equal(200))
					Ω(resp.Header).Should(HaveKey("Access-Control-Allow-Headers"))
					Ω(resp.Header["Access-Control-Allow-Headers"]).Should(Equal([]string{header}))
				})

			})

		})
	})
})
