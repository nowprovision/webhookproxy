# WebHookProxy

** This is no longer maintained/supported, be aware race conditions are highly likely to exist in this code **

## Introduction

This is the core webhookproxy library providing an inversion of push/pull model of webhooks.

This works in conjuction with webhookproxy-saas and
webhookproxyweb, all open source.

# Motivation

I used a fleeting frustration with Google Drive SDK webhooks to get used to the Go Programming language again 
(long time absent, and a long way to go still), the outcome was essentially this project.

## Write clients, not servers

Servers for web hooks (aka postbacks) often annoy me, you need some compute resources with a routable IP address, 
and quite often a non self-signed SSL certificate. For development this is particuarly annoying, and even 
though services like [localtunnel.me](http://www.localtunnel.me) ease the burden, one still needs to deal 
with embedding a web server in our applications. 

WebHookProxy accepts web hook payloads, and then allows your application to connect as a http client to
pick up the web hook payloads, and via another http client call return a response. It inverts the push/pull
model. Run it yourself via webhookproxy-single,
webhookproxy-saas,
or eventually use free hosted SaaS.

## How it works

You point the web hook callee such as Google Drive API postback to your webhookproxy instance such as 
https://acmecorp.webhookproxy.com/webhook/swordfish where swordfish is a choosen secret and acmecorp
is a choosen subdomain

When the web hook callee sends a payload via POST (e.g. new file in google drive, new transaction in paypal), webhookproxy
accepts the HTTP request (but does not complete it yet), and queues the task of reading the body of the request. 
The request is currently in limbo.

Once a client connects to /poll/swordfish via GET side the requests are married up and the request payload of the web hook callee
becomes the response to the client, a X-Replyid is also provides in the response headers. 
Clients are expected to long-poll /poll/swordfish so there's also someone waiting (as webhook
callee may timeout otherwise), for a large number of postbacks you can have several long polling client targetting
the same endpoint..

Once the client processes the payload it returns a response at /reply/swordish via POST (using the X-ReplyTo-Id header with
value provided by X-Replyid in the poll request), which is then returned to web hook callee as its response.

Operating in long poll mode latency overhead is minimal, one can check latency overhead via the X-Whproxy-Delay HTTP header 
on the poll request.

## Better explained with cURL

    (1) -> curl -vvv -X POST -d 'Hello you have a new file' https://myfilesync.webhookproxy.com/webhook/swordfish 

    (2) -> curl -vvv -X GET https://myfilesync.webhookproxy.com/poll/swordfish 

    (2) <-
    < HTTP/1.1 200 OK
    < Connection: keep-alive
    < X-Replyid: 0a6d1ecb-e538-4bcc-accc-5d57fb0d7f84
    < X-Whdelaysecs: 0.12126
    < X-Whfrom: 127.0.0.1:33042
    < X-Whheader-Accept: */*
    < X-Whheader-Connection: close
    < X-Whheader-User-Agent: curl/7.43.0
    < X-Whheader-X-Forwarded-For: 1.1.1.1
    < 
    * Connection #0 to host myfilesync.webhookproxy.com left intact
    Hello you have a new file%
    
    (3) -> curl -vvv -X POST -d "File received" -H 'X-InReplyTo: 0a6d1ecb-e538-4bcc-accc-5d57fb0d7f84' https://myfilesync.webhookproxy.com/reply/swordfish 

    (3) <- HTTP/1.1 200 OK

    (1) <- HTTP/1.1 200 OK
    < Connection: keep-alive

    * Connection #0 to host myfilesync.webhookproxy.com left intact
    File received%

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

The project webhookproxy-single is a very simple host, as suggested by the name it only supports one
webhook configuration, which is kind of defeating the point if we do aim to avoid wasting time with endpoint setup.

The gain any real benefit, as in not setting up endpoints and SSL certs per web hook, then it needs 
to handle multiple distinct endpoints, so a URL prefix per webhook setup or subdomain under a wildcard SSL host needs implementing,webhookproxy-saas provides a Saas implementation.

## SaaS - Free hosted SSL webhook endpoints

I am in process of setting up SaaS to provide free SSL web hook endpoints and proxying.

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

