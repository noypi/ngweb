ngweb
=====

A Tiny Web Server for Static Pages

See [An example of running the AngularJS Documentation locally](http://ngtutorial.com/tool/tiny-web-server-for-static-pages.html)

# Example configuration

```toml
# add ng to "/etc/hosts"
# ... example:
# ...
# 127.0.1.1	ng
# 127.0.1.1	ngtutorial.com
# ...
# for windows:
# %system32%/driver/etc/hosts
servername = "ng"
port = "80"

#
# enable HTTPS, set tls = true
# tls = false
#
# specify the certificate file for HTTPS connections.
# certfile = "./priv/cert.pem"
#
# specify the key file for HTTPS connections.
# keyfile = "./priv/key.pem"
#

[[route]]
pattern = "/(api|guide|misc|tutorial|error)"
path =  "/d/dev/js/angular-1.3.0/docs/index.html"
filealias = true
priority = 20

[[route]]
pattern = "/angular.*js"
path =  "/d/dev/js/angular-1.3.0/"
priority = 10

[[route]]
pattern = "/" 
path =  "/d/dev/js/angular-1.3.0/docs"
priority = 0


# 
# ngweb -genconfig > config.toml
#

```