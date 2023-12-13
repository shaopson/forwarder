forwarder
=============
a reverse proxy, port forwarding tool. support tcp, udp proxy, stream encryption.


## Usage
download: [Release](https://github.com/shaopson/forwarder/releases/)
### Server side
config file
```
# forwarder server config

# bind_ip = 0.0.0.0
bind_port = 8848

# client auth token
# auth_token = 

# log file path
# log_file = /var/log/forwarder_8848.log
```

start service
```shell
forwarder-server -c forwarder.server.conf
```

### Client side
config file
```
# forwarder client config
[server]
server_ip = 
server_port = 8848

# auth token
# auth_token =

# log file path
# log_file = /var/log/forwarder_c.log

# proxy config
[ssh]
type = tcp
local_ip = 127.0.0.1
local_port = 22
# remote_ip =
remote_port = 8000
```
start client
```
forwarder-cli -c forwarder.client.conf
```
The above is an example of an ssh proxy. The client registers a tcp proxy on port 8000 with the server, and the proxy forwards the data stream to the client's 127.0.0.1:22 address

Don't forget to open the ports in your firewall.
