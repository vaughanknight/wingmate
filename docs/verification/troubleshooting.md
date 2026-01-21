# Wingmate Troubleshooting Guide

This guide covers common issues when setting up and running Wingmate agents.

## Connection Issues

### Connection Refused

**Symptom**:
```
Ping failed: dial tcp 192.168.1.100:9001: connect: connection refused
```

**Causes**:
1. Agent not running on target machine
2. Wrong port number
3. Agent bound to localhost only

**Solutions**:
1. Verify agent is running: `ps aux | grep wingmate`
2. Check port: Agent should show `listening on http://0.0.0.0:<port>`
3. Restart with correct port: `./wingmate --name agent --port 9001`

### Connection Timeout

**Symptom**:
```
Ping failed: dial tcp 192.168.1.100:9001: i/o timeout
```

**Causes**:
1. Firewall blocking port
2. Network routing issues
3. Machine not reachable

**Solutions**:

**Linux (ufw)**:
```bash
# Check status
sudo ufw status

# Allow port
sudo ufw allow 9001/tcp
```

**Linux (iptables)**:
```bash
# Allow incoming
sudo iptables -A INPUT -p tcp --dport 9001 -j ACCEPT
```

**macOS**:
- System Preferences > Security & Privacy > Firewall > Firewall Options
- Add wingmate to allowed applications

**Windows**:
```powershell
# Allow port
netsh advfirewall firewall add rule name="Wingmate" dir=in action=allow protocol=TCP localport=9001
```

### DNS Resolution Failed

**Symptom**:
```
Ping failed: dial tcp: lookup machineA: no such host
```

**Solutions**:
1. Use IP address instead of hostname
2. Add entry to `/etc/hosts`:
   ```
   192.168.1.100 machineA
   ```
3. Configure DNS server

## Port Issues

### Port Already in Use

**Symptom**:
```
Failed to start agent: listen tcp :9001: bind: address already in use
```

**Solutions**:
1. Use different port: `./wingmate --port 9002`
2. Find and stop process using port:
   ```bash
   # Find process
   lsof -i :9001

   # or
   netstat -tlnp | grep 9001

   # Kill process
   kill <pid>
   ```

### Permission Denied (Ports < 1024)

**Symptom**:
```
Failed to start agent: listen tcp :80: bind: permission denied
```

**Solution**:
Use ports >= 1024, or run as root (not recommended):
```bash
./wingmate --port 9001  # Use high port instead
```

## Protocol Issues

### Invalid Response

**Symptom**:
```
Ping failed: peer is not an A2A agent
```

**Causes**:
1. Wrong service running on port
2. Agent misconfigured
3. Proxy intercepting traffic

**Solutions**:
1. Verify Agent Card:
   ```bash
   curl http://192.168.1.100:9001/.well-known/agent.json
   ```
2. Check for proxy headers in response
3. Ensure no reverse proxy is modifying requests

### JSON Parse Error

**Symptom**:
```
Failed to get Agent Card: invalid JSON response
```

**Solutions**:
1. Check raw response:
   ```bash
   curl -v http://192.168.1.100:9001/.well-known/agent.json
   ```
2. Look for HTML error pages or proxy messages

## Configuration Issues

### Name Required

**Symptom**:
```
Error: --name is required for server mode
```

**Solution**:
```bash
./wingmate --name my-agent --port 9001
```

### Invalid Config File

**Symptom**:
```
Configuration error: load config: invalid JSON
```

**Solutions**:
1. Validate JSON:
   ```bash
   cat config/agent.json | jq .
   ```
2. Check for trailing commas
3. Ensure proper quoting

## Flight Log Issues

### Log File Not Created

**Symptom**: No `flight.jsonl` file after sending messages

**Solutions**:
1. Check permissions on directory
2. Specify explicit path: `./wingmate --log /tmp/flight.jsonl`
3. Check disk space

### Log File Permissions

**Symptom**:
```
Failed to create agent: open flight.jsonl: permission denied
```

**Solutions**:
1. Check directory permissions
2. Run from writable directory
3. Specify writable path: `./wingmate --log ~/flight.jsonl`

## Network Diagnostics

### Test Network Connectivity

```bash
# Ping (ICMP)
ping 192.168.1.100

# Check if port is open (from remote)
nc -zv 192.168.1.100 9001

# or
telnet 192.168.1.100 9001

# Check local listening ports
netstat -tlnp
ss -tlnp
```

### Check HTTP Connectivity

```bash
# Basic HTTP check
curl -v http://192.168.1.100:9001/.well-known/agent.json

# With timeout
curl --connect-timeout 5 http://192.168.1.100:9001/.well-known/agent.json
```

### Trace Route

```bash
traceroute 192.168.1.100
# or
mtr 192.168.1.100
```

## Quick Reference

| Symptom | Likely Cause | Quick Fix |
|---------|--------------|-----------|
| Connection refused | Agent not running | Start agent |
| Connection timeout | Firewall | Open port |
| DNS failed | Name not resolvable | Use IP address |
| Port in use | Another process | Use different port |
| Permission denied | Low port number | Use port >= 1024 |
| Invalid response | Wrong service | Check Agent Card URL |
| Name required | Missing --name | Add --name flag |
| Config invalid | Bad JSON | Validate with jq |

## Getting Help

If issues persist:

1. Run with `--verbose` flag for detailed output
2. Check Flight Log for errors
3. Capture curl output with `-v` flag
4. Report issue with:
   - OS and version
   - Wingmate version
   - Command used
   - Full error message
   - Network configuration (redacted IPs OK)
