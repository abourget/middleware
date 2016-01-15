# goa Middlewares

This repository contains middlewares for the [goa](http://goa.design) web application framework.

[![Build Status](https://travis-ci.org/raphael/goa-middleware.svg?branch=master)](https://travis-ci.org/raphael/goa-middleware)

Each middleware is provided as a separate Go package.

#### JWT

Package [jwt](https://godoc.org/github.com/raphael/goa-middleware/jwt) contributed by @bketelsen
adds the ability for goa services to use [JSON Web Token](http://jwt.io/) authorization.

#### CORS

Package [cors](https://godoc.org/github.com/raphael/goa-middleware/cors) adds
[Cross Origin Resource Sharing](https://en.wikipedia.org/wiki/Cross-origin_resource_sharing) support
to goa services.

#### Gzip

Package [gzip](https://godoc.org/github.com/raphael/goa-middleware/gzip) contributed by @tylerb adds the ability to compress response bodies using gzip format as specified in RFC 1952.

#### Defer Panic

Package [dpgoa/middleware](https://godoc.org/github.com/deferpanic/dpgoa/middleware) contributed
by [Defer Panic](https://github.com/deferpanic) adds the ability for goa services to leverage the
[Defer Panic service](https://deferpanic.com/).

