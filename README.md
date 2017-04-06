go-xmpp
=======

go xmpp library

currently used for Openfire load test. 

Test environment configuration:
```
Server: jabber.hylaa.net:5222
pre-created testing accounts: u_1, u_2, ... , u_60 
testing accounts password: P@ssw0rd
All testing accounts are under the same group (roster) so they can talk to each other.
```

The binary file "go-xmpp" is built for Linux amd64 OS.

For help 
```bash
$ ./go-xmpp -h
```

Example: Of all the 60 users, each user on average sends 1 message per 100 milliseconds, and sends 50 messages in total
```bash
$ ./go-xmpp -f 100 -t 50
```