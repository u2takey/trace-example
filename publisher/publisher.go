package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/u2takey/trace-example/lib/mq"
	"github.com/u2takey/trace-example/lib/tracing"
)

var tracer, _ = tracing.Init("publisher")

func main() {
	http.HandleFunc("/publish", func(w http.ResponseWriter, r *http.Request) {
		spanCtx, _ := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header))
		span := tracer.StartSpan("publisher", ext.RPCServerOption(spanCtx))
		defer span.Finish()

		helloStr := r.FormValue("helloStr")
		println(helloStr)

		go sendMsg(opentracing.ContextWithSpan(context.Background(), span), helloStr)
	})

	log.Fatal(http.ListenAndServe(":8082", nil))
}

var (
	amqpURI = "amqp://root:123456@localhost:5672"
	routingKey = ""
)


func sendMsg(ctx context.Context, m string){

	span, _ := opentracing.StartSpanFromContextWithTracer(ctx, tracer, "callback")
	defer span.Finish()

	v := url.Values{}
	v.Set("helloStr", m)
	ext.SpanKindRPCClient.Set(span)
	ext.PeerAddress.Set(span, amqpURI)

	c := map[string]string{}
	span.Tracer().Inject(span.Context(), opentracing.HTTPHeaders, opentracing.TextMapCarrier(c))
	fmt.Println("inject", c)
	err := mq.Publish(amqpURI, routingKey, m, false, c)
	if err != nil{
		fmt.Println("publish error", err)
	}
}