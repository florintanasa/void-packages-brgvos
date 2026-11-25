Before starting the 'rei3' runit service, review /etc/rei3/config.json and
/etc/default/rei3:

1. Database connection:
   On Linux, REI3 uses a dedicated PostgreSQL server. Set "embedded": false
   in /etc/rei3/config.json and configure your PostgreSQL host, port, database
   name, user and password (a UTF-8 database with full owner privileges).

2. Web server port:
   The service runs unprivileged as the '_rei3' user and cannot bind to ports
   below 1024 without capabilities. Set "port": 8443 (or another port > 1024)
   in /etc/rei3/config.json, then access https://localhost:8443 (default login:
   admin / admin).

3. Custom data directory (optional):
   To store REI3 data under /srv/rei3 instead of /var/lib/rei3, set
   REI3_ROOT="/srv/rei3" in /etc/default/rei3 before enabling the service.
