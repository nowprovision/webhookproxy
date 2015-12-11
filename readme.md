# WebHookProxy

## Introduction

This is the core webhookproxy library providing an inversion of push/pull model of webhooks.

This works in conjuction with [webhookproxy-saas](http://www.github.com/nowprovision/webhookproxy-saas) and
[webhookproxyweb](http://www.github.com/nowprovision/webhookproxyweb), all open source.

# Motivation

I used a fleeting frustration with Google Drive SDK webhooks to get used to the Go Programming language again 
(long time absent, and a long way to go still), the outcome was essentially this project.

## Write clients, not servers

Servers for web hooks often annoy me, you need some compute resources with a routable IP address, 
and quite often a non self-signed SSL certificate. For development this is particuarly annoying, and even 
though services like [localtunnel.me](http://www.localtunnel.me) ease the burden, one still needs to deal 
with embedding a web server in our applications. 

WebHookProxy accepts web hook payloads, and then allows your application to connect as a http client to
pick up the web hook payloads, and via another http client call return a response. It inverts the push/pull
model. Run it yourself via [webhookproxy-single](http://www.github.com/nowprovision/webhookproxy-single) or use our upcoming free SaaS.

## How it works

You point the web hook callee such as Google Drive to your webhookproxy instance (e.g. set URL to http(s)://fqdn/prefix/webhook).

When the web hook callee sends a payload (e.g. new file in google drive, new transaction in paypal), webhookproxy
accepts the HTTP request, and queues the task of reading the body of the request.

Once a client connects to /poll side the requests are married up and the request payload of the web hook callee
becomes the response to the client.

Once the client processes the payload it returns a response at /reply (using the X-ReplyTo-Id of the /poll response) 
which is then returned to web hook callee as the response.

Operating in long poll mode latency overhead is minimal, though please check the X-Whproxy-Delay HTTP header.

## Enterprise

What about persistence?

If no client connects to /poll within a period of time we'll respond with a try again later code (typically 500 or 503) 
telling the web hook callee we have not processed the request. Any reasonable hook callee will follow convention and
try again later when response is not 20x, possibly using an expoential backoff like Google, until it gets a 200 response. 

So Janice in accounting says it's okay..

## Data agnostic

Webhookproxy simply inverts the push / pull notion, by proxying the streams to from the webhook to
the http client, and back from second http call to the webhook callee as a reply. No inspection
is done, any Content-Type are fine, any arbitary payload should work, form urlencoded, json, binary,yaml etc..



## Self-Hosting and Deployment 

The project [webhookproxy-single](http://www.github.com/nowprovision/webhookproxy-single) is a very simple host, as suggested by the name it only supports one
webhook configuration, which is kind of defeating the point if we do aim to avoid wasting time with endpoint setup.

The gain any real benefit, as in not setting up endpoints and SSL certs per web hook, then it needs 
to handle multiple distinct endpoints, so a URL prefix per webhook setup or subdomain under a wildcard SSL host needs implementing,
[webhookproxy-saas](http://www.github.com/nowprovision/webhookproxy-saas) provides a Saas implementation.

## SaaS - Free hosted SSL webhook endpoints

I am in process of setting up [https://www.webhookproxy.com](https://www.webhookproxy.com) to provide free SSL web hook endpoints and proxying.

A simple SaaS, giving you a custom subdomain proteted by a wildcard SSL certificate, letting you call to 
pickup the requests and provide responses. 

Plan is to also provide site verification DNS records for example using a webhook with Google Drive SDK.  

Note: Site is up and running, but product is not yet working! 


## Disclaimer

Minimal viable product. I know parts of my go lang code are less than optimal to put it politely, this is a work in progress
and an area of self-improvement I am working on.

PR, code review, suggestions etc.. welcome, full credit will be given.

## License

MIT

## Author

Matt Freeman - [@nowprovision](http://www.twitter.com/nowprovision)

