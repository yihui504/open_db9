# Incident Response Runbook

## 🚨 Incident Severity Levels

| Level | Severity | Response Time | Description |
|-------|----------|---------------|-------------|
| P0 | **Critical** | Immediate (15 min) | Complete system outage, data loss, security breach |
| P1 | **High** | 1 hour | Major functionality impaired, affecting all users |
| P2 | **Medium** | 4 hours | Partial functionality issues, affecting some users |
| P3 | **Low** | 8 hours | Minor issues, cosmetic problems, limited impact |
| P4 | **Info** | 24 hours | Informational, maintenance notifications |

## 📞 On-Call Procedures

### Initial Response
1. **Acknowledge Incident**: All on-call must acknowledge page within 15 minutes
2. **Create Incident Ticket**: Create ticket in Linear with proper severity
3. **Assemble Team**: Notify relevant team members
4. **Communication**: Post incident channel update

### Communication Protocol
- **Primary Channel**: Slack #incidents
- **Updates**: Every 30 minutes if no resolution
- **Escalation**: Notify manager after 2 hours unresolved

## 🔍 Common Incidents

### 1. Database Connection Failures

**Symptoms**:
- API 503 errors
- Connection timeouts
- CLI authentication failures

**Immediate Actions**:
1. Check database service status
2. Verify network connectivity
3. Check credentials in vault

**Checklist**:
- [ ] Database process running
- [ ] Network connectivity (`pg_isready`)
- [ ] Disk space (>20% free)
- [ ] Authentication server accessible

**Resolution**:
- Restart database service if needed
- Restore from backup if corruption detected
- Failover to standby if primary unavailable

### 2. High Memory/CPU Usage

**Symptoms**:
- Slow API responses
- Query timeouts
- High memory (>90%)
- CPU spikes

**Immediate Actions**:
1. Identify resource-intensive queries
2. Check for connection leaks
3. Review query plans

**Checklist**:
- [ ] Run `pg_stat_activity` to find active queries
- [ ] Check for long-running transactions
- [ ] Verify connection pool settings
- [ ] Review query performance metrics

**Resolution**:
- Kill problematic queries
- Optimize slow queries
- Adjust connection pool size
- Scale resources if needed

### 3. Authentication Failures

**Symptoms**:
- 401/403 errors on API calls
- CLI login failures
- JWT token errors

**Immediate Actions**:
1. Check auth service status
2. Verify JWT secret
3. Check user database

**Checklist**:
- [ ] Authentication service running
- [ ] JWT secret valid
- [ ] Database accessible
- [ ] User accounts active

**Resolution**:
- Restart auth service
- Rotate JWT secrets if compromised
- Reset user passwords if needed
- Enable verbose logging for debugging

### 4. Rate Limit Exhaustion

**Symptoms**:
- 429 errors
- API throttling
- Slow response times

**Immediate Actions**:
1. Check rate limit metrics
2. Identify abusive clients
3. Adjust limits if needed

**Checklist**:
- [ ] Monitor rate limit dashboard
- [ ] Check for unusual request patterns
- [ ] Verify client configurations
- [ ] Review API key usage

**Resolution**:
- Temporarily increase limits
- Block abusive IPs
- Implement client-side caching
- Scale rate limiting service

### 5. Slow Query Performance

**Symptoms**:
- Query timeouts
- High database load
- User complaints about slowness

**Immediate Actions**:
1. Identify slow queries
2. Check database indexes
3. Review query plans

**Checklist**:
- [ ] Enable query logging
- [ ] Run `pg_stat_statements`
- [ ] Check table statistics
- [ ] Verify index usage

**Resolution**:
- Add missing indexes
- Optimize query structure
- Update database statistics
- Increase query timeout settings
- Consider query caching

## 📈 Escalation Procedures

### First Escalation (After 2 hours)
- Notify Engineering Manager
- Engage senior on-call engineer
- Wider team notification

### Second Escalation (After 4 hours)
- Notify Director of Engineering
- Engage additional subject matter experts
- Consider customer communication

### Third Escalation (After 6 hours)
- Notify VP of Engineering
- Engage external support if needed
- Prepare customer status update

### Executive Escalation (After 8 hours)
- Notify Chief Technology Officer
- Prepare incident review meeting
- Consider rollback plans

## 📋 Post-Incident Review Template

### Incident Summary
- **Incident ID**: [Unique identifier]
- **Start Time**: [Timestamp]
- **End Time**: [Timestamp]
- **Duration**: [X hours Y minutes]
- **Impact**: [Users affected, business impact]
- **Severity**: [P0-P4]

### Root Cause Analysis
- **Primary Cause**: [Technical root cause]
- **Contributing Factors**: [Any contributing issues]
- **Detection**: [How incident was detected]
- **Containment**: [Actions taken to contain]

### Timeline
```markdown
00:00 - Incident detected
00:15 - Initial response team assembled
00:30 - Root cause identified
01:00 - Resolution implemented
01:30 - Service restored
02:00 - Post-mortem initiated
```

### Resolution Details
- **Solution**: [Steps taken to resolve]
- **Rollback Plan**: [If applicable]
- **Monitoring**: [Post-resolution monitoring]

### Action Items
- [ ] Short-term fixes (to be implemented within 1 week)
- [ ] Long-term improvements (to be implemented within 1 month)
- [ ] Process changes
- [ ] Documentation updates

### Customer Communication
- **Notification Time**: [When customers were notified]
- **Message**: [Communication sent to customers]
- **Follow-up**: [Any required follow-up]

### Learnings
- **What went well**: [Positive aspects of response]
- **Areas for improvement**: [Issues in response process]
- **Preventative measures**: [How to prevent recurrence]

### Sign-offs
- Incident Lead: [Name]
- Engineering Manager: [Name]
- Director of Engineering: [Name]

---

*This runbook should be reviewed quarterly and updated as needed*