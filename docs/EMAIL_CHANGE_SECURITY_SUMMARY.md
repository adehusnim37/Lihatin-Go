# Email Change Security Implementation Summary

## ✅ Implemented Features

### 1. **Rate Limiting (30 Days)**
- ⏱️ Maximum 1x email change per 30 hari
- 📊 Check history sebelum allow change email
- 🚫 Block attempt jika masih dalam 30 hari window

### 2. **Suspicious Activity Detection**
- 🚨 Auto-lock jika ada `email_change_revoked` dalam 30 hari
- 🔒 Prevent abuse setelah revoke (possible account compromise)
- ⚠️ Different error response untuk rate limit vs lock

### 3. **Eligibility Check API**
- 👤 User dapat check sebelum attempt change
- 📅 Response include `days_remaining` untuk countdown
- 🎯 Frontend-friendly untuk UX

### 4. **History Tracking API**
- 📜 Get last 90 days email change history
- 🔍 Include all related actions (change, revoke, verification)
- 📊 Useful untuk user audit dan support

---

## 🔧 New Files Created

1. **`controllers/auth/email/check-eligibility.go`**
   - `CheckEmailChangeEligibility()` - Check if user can change email
   - `GetEmailChangeHistory()` - Get user's email history

---

## 📝 Modified Files

### 1. **`utils/app_error.go`**
Added new errors:
- `ErrEmailChangeRateLimitExceeded` (429) - Max 1x per 30 days
- `ErrEmailChangeLocked` (403) - Locked due to suspicious activity

### 2. **`repositories/user-auth-repo.go`**
Updated `ChangeEmail()`:
- Check history 30 hari terakhir
- Block if `email_change_revoked` found (suspicious)
- Block if `email_change` found (rate limit)

Added new functions:
- `CheckEmailChangeEligibility()` - Helper untuk check eligibility
- `GetEmailChangeHistory()` - Get history untuk analysis

### 3. **`routes/auth_routes.go`**
Added new routes:
- `GET /auth/check-email-change-eligibility`
- `GET /auth/email-change-history`

### 4. **`EMAIL_CHANGE_FLOW.md`**
Updated documentation:
- Added rate limiting section
- Added security flow diagram
- Added test cases untuk rate limit & lock
- Added example scenarios

---

## 🎯 How It Works

### Rate Limit Flow
```go
User Request Change Email
         ↓
Query history_users WHERE 
  - user_id = ?
  - action_type IN ('email_change', 'email_change_revoked')
  - changed_at > (NOW() - 30 days)
         ↓
    Found History?
         ↓
    ┌────┴────┐
    │         │
   YES       NO
    │         │
    ↓         ↓
Check Type   ALLOW
    │
    ├─ email_change? → 429 Rate Limit
    │
    └─ email_change_revoked? → 403 Locked
```

### Database Query
```sql
SELECT * FROM history_users
WHERE user_id = ?
  AND action_type IN ('email_change', 'email_change_revoked')
  AND changed_at > NOW() - INTERVAL '30 days'
ORDER BY changed_at DESC
LIMIT 1;
```

---

## 🧪 Test Scenarios

### Scenario 1: Normal Rate Limit
```bash
# Day 0
POST /auth/change-email {"new_email": "new@email.com"}
# → Success

# Day 15
POST /auth/change-email {"new_email": "another@email.com"}
# → Error 429: "You can only change your email once every 30 days"
# → days_remaining: 15
```

### Scenario 2: Suspicious Lock
```bash
# Day 0
POST /auth/change-email {"new_email": "new@email.com"}
# → Success

# Day 1
GET /auth/verify-email?token=xxx
# → Success (email verified)

# Day 2
GET /auth/revoke-email-change?token=xxx
# → Success (revoked back to old email)

# Day 3
POST /auth/change-email {"new_email": "hack@email.com"}
# → Error 403: "Email change is temporarily locked due to suspicious activity"
```

### Scenario 3: Check Eligibility
```bash
# After change email
GET /auth/check-email-change-eligibility
# Response:
{
  "eligible": false,
  "days_remaining": 20,
  "message": "You must wait before changing your email again",
  "reason": "rate_limit"
}

# After 30 days
GET /auth/check-email-change-eligibility
# Response:
{
  "eligible": true,
  "days_remaining": 0,
  "message": "You are eligible to change your email"
}
```

---

## 🚀 Frontend Integration

### Before Change Email Form
```typescript
// Check eligibility first
const checkEligibility = async () => {
  const response = await fetch('/api/v1/auth/check-email-change-eligibility', {
    headers: { Authorization: `Bearer ${token}` }
  });
  const data = await response.json();
  
  if (!data.data.eligible) {
    if (data.data.reason === 'rate_limit') {
      showError(`You can change email in ${data.data.days_remaining} days`);
    } else {
      showError('Email change is locked due to suspicious activity');
    }
    disableChangeEmailButton();
  }
};
```

### Show History
```typescript
const getHistory = async () => {
  const response = await fetch('/api/v1/auth/email-change-history', {
    headers: { Authorization: `Bearer ${token}` }
  });
  const data = await response.json();
  
  data.data.history.forEach(item => {
    console.log(`${item.action_type} at ${item.changed_at}`);
  });
};
```

---

## 🔐 Security Benefits

1. **Prevent Abuse:**
   - Rate limit stops attacker dari multiple change attempts
   - 30 days window cukup untuk legitimate user detect issue

2. **Account Compromise Detection:**
   - Revoke = red flag untuk possible compromise
   - Auto-lock gives legitimate owner time untuk secure account

3. **Audit Trail:**
   - Full history untuk forensic analysis
   - Support team dapat investigate suspicious patterns

4. **User Protection:**
   - Proactive blocking sebelum damage occurs
   - Clear error messages untuk user understanding

---

## 📊 Metrics to Monitor

1. **Rate Limit Hits:**
   - Track berapa user hit rate limit
   - Possible indicator untuk UX issues

2. **Suspicious Locks:**
   - Track berapa account locked after revoke
   - High number = possible attack patterns

3. **History Queries:**
   - Track API usage untuk performance tuning
   - Cache considerations untuk frequent checks

---

## 🎓 Best Practices

1. **Always Check Eligibility First:**
   ```go
   eligible, daysRemaining, err := repo.CheckEmailChangeEligibility(userID)
   if !eligible {
       return fmt.Errorf("must wait %d days", daysRemaining)
   }
   ```

2. **Log All Security Events:**
   ```go
   utils.Logger.Warn("Rate limit exceeded",
       "user_id", userID,
       "days_remaining", daysRemaining,
   )
   ```

3. **Clear Error Messages:**
   ```go
   return utils.ErrEmailChangeRateLimitExceeded // HTTP 429
   return utils.ErrEmailChangeLocked            // HTTP 403
   ```

4. **Transaction Safety:**
   - All database operations dalam transaction
   - Rollback on any error
   - Atomic state changes

---

## 🔄 Migration Notes

### No Database Changes Required
- Uses existing `history_users` table
- Only query logic added
- Backward compatible

### Deployment Steps
1. Deploy code changes
2. Test eligibility check endpoint
3. Test rate limiting dengan mock dates
4. Monitor logs untuk error rates
5. Update frontend dengan eligibility check

---

## 📞 Support Scenarios

### User: "Why can't I change my email?"

**Check:**
```bash
# As admin
GET /api/v1/auth/email-change-history?user_id=xxx
```

**Response Patterns:**

1. **Recent change found:**
   - "You changed email 10 days ago, please wait 20 more days"

2. **Recent revoke found:**
   - "Your account was flagged for suspicious activity. For security, email changes are locked for 30 days"
   - Escalate to security team

3. **No history:**
   - Check other errors (not verified, etc)

---

## ✨ Future Enhancements

1. **Whitelist Trusted Users:**
   - Premium users = no rate limit
   - Verified business accounts = shorter wait

2. **Admin Override:**
   - Admin dapat force-enable email change
   - Requires admin approval + reason

3. **Grace Period:**
   - 1 free change per year
   - Additional changes = wait 30 days

4. **Email Notification:**
   - Notify user when eligible again
   - "You can now change your email"

---

## 🎯 Success Metrics

After implementation, expect:
- ✅ 90% reduction dalam email change abuse
- ✅ Faster account compromise detection
- ✅ Better user trust (visible security measures)
- ✅ Reduced support tickets (self-service eligibility check)

---

**Implementation Date:** October 11, 2025  
**Status:** ✅ Complete & Ready for Testing  
**Breaking Changes:** None (backward compatible)
