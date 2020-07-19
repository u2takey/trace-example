package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
	"github.com/u2takey/trace-example/lib/http"
	"github.com/u2takey/trace-example/lib/mq"
	"github.com/u2takey/trace-example/lib/tracing"
)


func main() {
	tracer, close := tracing.Init("hello-world")
	defer close.Close()
	opentracing.SetGlobalTracer(tracer)

	helloTo := "jack"
	greeting := "tom"

	span := tracer.StartSpan("say-hello")
	span.SetTag("hello-to", helloTo)
	span.SetBaggageItem("greeting", greeting)

	ctx := opentracing.ContextWithSpan(context.Background(), span)

	helloStr := formatString(ctx, helloTo)
	printHello(ctx, helloStr)
	span.Finish()

	waitingMsg()
}

func formatString(ctx context.Context, helloTo string) string {
	span, _ := opentracing.StartSpanFromContext(ctx, "formatString")
	defer span.Finish()

	v := url.Values{}
	v.Set("helloTo", helloTo)
	url := "http://localhost:8081/format?" + v.Encode()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err.Error())
	}

	ext.SpanKindRPCClient.Set(span)
	ext.HTTPUrl.Set(span, url)
	ext.HTTPMethod.Set(span, "GET")
	span.Tracer().Inject(
		span.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(req.Header),
	)

	resp, err := xhttp.Do(req)
	if err != nil {
		panic(err.Error())
	}

	helloStr := string(resp)

	span.LogFields(
		log.String("event", "string-format"),
		log.String("value", helloStr),
	)

	return helloStr
}

func printHello(ctx context.Context, helloStr string) {
	span, _ := opentracing.StartSpanFromContext(ctx, "printHello")
	defer span.Finish()

	v := url.Values{}
	v.Set("helloStr", helloStr)
	url := "http://localhost:8082/publish?" + v.Encode()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err.Error())
	}

	ext.SpanKindRPCClient.Set(span)
	ext.HTTPUrl.Set(span, url)
	ext.HTTPMethod.Set(span, "GET")
	span.Tracer().Inject(span.Context(), opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(req.Header))

	if _, err := xhttp.Do(req); err != nil {
		panic(err.Error())
	}
}


var (
	amqpURI = "amqp://root:123456@localhost:5672"
)


func waitingMsg(){
	consumer, err := mq.NewConsumer(amqpURI, "test", "", "")
	if err != nil{
		fmt.Println("NewConsumer error", err)
	}

	for a := range consumer.Deliveries{
		msg := mq.Unmarshal(a.Body)
		func (){
			spanCtx, _ := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, opentracing.TextMapCarrier(msg.Extra))
			span := opentracing.GlobalTracer().StartSpan("got_callback", ext.RPCServerOption(spanCtx))
			defer span.Finish()
			span.LogKV("got", msg.Body)
		}()
		fmt.Println("got",msg.Body, msg.Extra)
		//break
		a.Ack(false)
		break
	}
}
