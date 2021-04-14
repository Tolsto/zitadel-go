package http

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/caos/oidc/pkg/client/rs"

	"github.com/caos/zitadel-go/pkg/api/middleware"
)

type IntrospectionInterceptor struct {
	resourceServer rs.ResourceServer
	handler        http.Handler
	marshaller     Marshaller
}

type Marshaller interface {
	Marshal(interface{}) ([]byte, error)
	ContentType() string
}

type JSONMarshaller struct{}

func (j JSONMarshaller) Marshal(i interface{}) ([]byte, error) {
	return json.Marshal(i)
}

func (j JSONMarshaller) ContentType() string {
	return "application/json"
}

//NewIntrospectionInterceptor intercepts every call and checks for a correct Bearer token using OAuth2 introspection
//(sending the token to the introspection endpoint)
func NewIntrospectionInterceptor(issuer, keyPath string) (*IntrospectionInterceptor, error) {
	resourceServer, err := rs.NewResourceServerFromKeyFile(issuer, keyPath)
	if err != nil {
		return nil, err
	}
	return &IntrospectionInterceptor{
		resourceServer: resourceServer,
		marshaller:     &JSONMarshaller{},
	}, nil
}

//Handler creates a http.Handler for middleware usage
func (interceptor *IntrospectionInterceptor) Handler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := interceptor.introspect(r)
		if err != nil {
			interceptor.writeError(w, 401, err.Error())
			return
		}
		handler.ServeHTTP(w, r)
	})
}

func (interceptor *IntrospectionInterceptor) introspect(r *http.Request) error {
	auth := r.Header.Get("authorization")
	if auth == "" {
		return fmt.Errorf("auth header missing")
	}
	return middleware.Introspect(r.Context(), auth, interceptor.resourceServer)
}

func (interceptor *IntrospectionInterceptor) writeError(w http.ResponseWriter, status int, errMessage interface{}) {
	b, err := interceptor.marshaller.Marshal(errMessage)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("content-type", interceptor.marshaller.ContentType())
	w.WriteHeader(status)
	_, err = w.Write(b)
	if err != nil {
		log.Println("error writing response")
	}
}
