# Email Change Flow Documentation

## 📋 Overview
Flow lengkap untuk proses change email dengan notifikasi ke old dan new email, termasuk mekanisme revoke untuk keamanan.

## 🔄 Complete Flow

### 0. **Check Eligibility (Optional)**
**Endpoint:** `GET /api/v1/auth/check-email-change-eligibility`

**Response:**
```json
{
  "success": true,
  "data": {
    "eligible": true,
    "days_remaining": 0,
    "message": "You are eligible to change your email"
  }
}
```

OR (if not eligible):
```json
{
  "success": true,
  "data": {
    "eligible": false,
    "days_remaining": 15,
    "message": "You must wait before changing your email again",
    "reason": "rate_limit"
  }
}
```

---

### 1. **User Request Change Email**
**Endpoint:** `POST /api/v1/auth/change-email`

**Request Body:**
```json
{
  "new_email": "newemail@example.com"
}
```

**Backend Process:**
- Validasi user sudah login dan email verified
- **🔒 Check email change history 30 hari terakhir:**
  - Jika ada `email_change_revoked` dalam 30 hari → **LOCK** (suspicious activity)
  - Jika ada `email_change` dalam 30 hari → **RATE LIMIT** (max 1x per 30 hari)
- Check new email belum digunakan user lain
- Generate revoke token (valid 24 jam)
- **Simpan history** dengan:
  - `old_value`: `{"email": "oldemail@example.com"}`
  - `new_value`: `{"email": "newemail@example.com"}`
  - `revoke_token`: token untuk undo
  - `revoke_expires`: 24 jam dari sekarang
- **Update email** user ke new email
- **Set `is_email_verified = false`**
- Generate verification token untuk new email
- **Kirim email ke NEW EMAIL** dengan link verifikasi

**Response:**
```json
{
  "success": true,
  "message": "Change email request processed. Please verify your new email address as soon as possible."
}
```

---

### 2. **User Verify New Email**
**Endpoint:** `GET /api/v1/auth/verify-email?token=xxx`

**Backend Process:**
- Validasi token belum expired
- Check `is_email_verified == false`
- **Set `is_email_verified = true`**
- Clear verification token
- Log ke history: `email_verification`
- **Ambil history email change terakhir** (untuk dapat old email dan revoke token)
- **Kirim 2 email:**
  1. **Ke NEW EMAIL:** Email success verification
  2. **Ke OLD EMAIL:** Notifikasi email changed + revoke link

**Response:**
```json
{
  "success": true,
  "message": "Email verification completed successfully"
}
```

**Email ke OLD EMAIL berisi:**
- Info: Email changed from `old@email.com` to `new@email.com`
- Button: "This Wasn't Me" (revoke link)
- Button: "This Was Me" (dashboard link)
- Warning: Link valid 24 jam
- Revoke URL: `https://backend.com/api/v1/auth/revoke-email-change?token=xxx`

---

### 3a. **User Click Revoke (This Wasn't Me)**
**Endpoint:** `GET /api/v1/auth/revoke-email-change?token=xxx`

**Backend Process:**
- Validasi revoke token belum expired
- Check action_type = `email_change`
- **Kembalikan email** ke old email
- **Set `is_email_verified = true`** (old email pasti sudah verified)
- Clear verification token
- **Mark revoke token as used** (set expires = now)
- Log ke history: `email_change_revoked`

**Response:**
```json
{
  "success": true,
  "message": "Email change has been revoked. Your original email has been restored and verified."
}
```

---

### 3b. **User Tidak Melakukan Apa-apa**
- Setelah 24 jam, revoke token expired
- Email change dianggap sah
- User tetap menggunakan new email

---

## 📊 Database Changes

### Table: `users`
```sql
email VARCHAR(255) -- Updated when change email requested
```

### Table: `user_auth`
```sql
is_email_verified BOOLEAN -- Set to false when change email, true after verify
email_verification_token VARCHAR(255) -- Token untuk verify new email
email_verification_token_expires_at TIMESTAMP
email_verification_source VARCHAR(50) -- 'signup' or 'change'
```

### Table: `history_users`
```sql
user_id VARCHAR(50)
action_type VARCHAR(50) -- 'email_change', 'email_verification', 'email_change_revoked'
old_value JSONB -- {"email": "old@example.com"}
new_value JSONB -- {"email": "new@example.com"}
revoke_token VARCHAR(100) -- Token untuk undo
revoke_expires TIMESTAMP -- 24 jam dari changed_at
reason TEXT
changed_at TIMESTAMP
changed_by VARCHAR(50)
ip_address VARCHAR(45)
user_agent TEXT
```

---

## 🔐 Security Features

1. **Rate Limiting (30 Days):**
   - ⏱️ Maximum 1x email change per 30 hari
   - 🚨 Auto-lock jika ada revoke dalam 30 hari (suspicious activity)
   - 📊 Check history sebelum allow change email
   - ⚠️ User dapat check eligibility sebelum attempt change

2. **Revoke Token:**
   - Valid 24 jam
   - One-time use (expired setelah digunakan)
   - Tied to specific email change history

3. **Email Verification Required:**
   - New email must be verified before notifications sent
   - Old email notification only sent after new email verified

4. **Transaction Safety:**
   - All operations use database transactions
   - Rollback on any error

5. **Audit Trail:**
   - Every action logged to history_users
   - Track IP address and user agent
   - Full history dapat di-query untuk analysis

---

## 🧪 Testing Checklist

### Test Case 1: Normal Flow
- [ ] User change email dari A ke B
- [ ] Email verifikasi terkirim ke B
- [ ] User verify email B
- [ ] Email notifikasi terkirim ke A dengan revoke link
- [ ] User di A click "This Was Me"
- [ ] Email tetap B, verified = true

### Test Case 2: Revoke Flow
- [ ] User change email dari A ke B
- [ ] User verify email B
- [ ] User di email A click "This Wasn't Me"
- [ ] Email kembali ke A, verified = true
- [ ] Revoke token marked as used

### Test Case 3: Expired Token
- [ ] User change email dari A ke B
- [ ] Wait 25 hours (atau manipulasi timestamp)
- [ ] User click revoke link
- [ ] Error: Token expired

### Test Case 4: Double Revoke
- [ ] User change email dan revoke
- [ ] Try to use same revoke token again
- [ ] Error: Token not found / expired

### Test Case 5: Verify Without History
- [ ] New user signup
- [ ] Verify email (no history yet)
- [ ] Should not fail, just no old email notification

### Test Case 6: Rate Limiting
- [ ] User change email (success)
- [ ] Wait 1 day
- [ ] Try to change email again → Error: Rate limit (29 days remaining)
- [ ] Check eligibility → eligible = false, days_remaining = 29
- [ ] Wait 30 days
- [ ] Try to change email again → Success

### Test Case 7: Suspicious Activity Lock
- [ ] User change email A → B
- [ ] Verify email B
- [ ] Revoke email change (back to A)
- [ ] Try to change email immediately → Error: Locked due to suspicious activity
- [ ] Check eligibility → eligible = false, reason = suspicious
- [ ] Wait 30 days
- [ ] Try to change email → Success

### Test Case 8: Security
- [ ] Try to change to existing email → Error
- [ ] Try to change to same email → Error
- [ ] Try to change without being verified → Error
- [ ] Try to change without login → 401 Unauthorized

---

## 📧 Email Templates

### 1. Change Email Verification (to NEW email)
- Subject: "Lihatin - Verify Your New Email"
- Content: Link to verify new email
- CTA: "Verify Email"

### 2. Email Change Notification (to OLD email)
- Subject: "Lihatin - Email Change Notification"
- Content: Info about email change
- CTA: "This Wasn't Me" (revoke) / "This Was Me" (dashboard)
- Valid: 24 hours

### 3. Success Verification (to NEW email)
- Subject: "Lihatin - Email Verified Successfully"
- Content: Confirmation email verified
- CTA: "Go to Dashboard"

---

## 🚨 Error Codes

| Code | Message | HTTP Status | When |
|------|---------|-------------|------|
| `EMAIL_CHANGE_RATE_LIMIT_EXCEEDED` | Can only change once every 30 days | 429 | User sudah change email dalam 30 hari |
| `EMAIL_CHANGE_LOCKED` | Locked due to suspicious activity | 403 | Ada revoke dalam 30 hari terakhir |
| `REVOKE_TOKEN_NOT_FOUND` | Revoke token not found | 400 | Token invalid atau sudah digunakan |
| `REVOKE_TOKEN_EXPIRED` | Revoke token has expired | 400 | Token > 24 jam |
| `INVALID_ACTION_TYPE` | Invalid action type | 400 | History bukan email_change |
| `INVALID_HISTORY_DATA` | Invalid history data | 500 | JSON corrupt |
| `EMAIL_ALREADY_VERIFIED` | Email already verified | 400 | Try verify 2x |
| `EMAIL_SAME_AS_CURRENT` | Email sama dengan current | 400 | New email = old email |
| `EMAIL_EXISTS` | Email sudah digunakan | 409 | Email conflict |
| `USER_EMAIL_NOT_VERIFIED` | Email belum verified | 403 | Try change email sebelum verify |

---

## 🔧 API Endpoints Summary

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| GET | `/api/v1/auth/check-email-change-eligibility` | ✅ Required | Check if user can change email |
| GET | `/api/v1/auth/email-change-history` | ✅ Required | Get user's email change history |
| POST | `/api/v1/auth/change-email` | ✅ Required | Request change email |
| GET | `/api/v1/auth/verify-email?token=xxx` | ❌ Public | Verify new email |
| GET | `/api/v1/auth/revoke-email-change?token=xxx` | ❌ Public | Revoke email change |

---

## �️ Rate Limiting & Security Logic

### Rate Limiting Rules (30 Days)

1. **Normal Email Change:**
   - User dapat change email maximum 1x per 30 hari
   - Counter dimulai dari `changed_at` di history
   - Setelah 30 hari, user eligible lagi

2. **Suspicious Activity Lock:**
   - Jika user melakukan revoke (`email_change_revoked`) dalam 30 hari
   - System auto-lock email change selama 30 hari dari revoke date
   - Reason: Possible account compromise atau abuse

3. **Eligibility Check:**
   - User dapat check eligibility sebelum attempt change
   - Response termasuk `days_remaining` jika tidak eligible
   - Frontend dapat show countdown timer

### Security Flow Diagram

```
User Request Change Email
         ↓
Check History (30 days)
         ↓
    ┌────┴────┐
    │         │
Found?    Not Found
    │         │
    ↓         ↓
    │      ALLOW
    │
    ├─ email_change? → RATE LIMIT (429)
    │
    └─ email_change_revoked? → LOCK (403)
```

### Example Scenarios

**Scenario 1: Normal Usage**
- Day 0: Change email A → B ✅
- Day 15: Try change B → C ❌ (15 days remaining)
- Day 30: Try change B → C ✅

**Scenario 2: Suspicious Activity**
- Day 0: Change email A → B ✅
- Day 1: Revoke to A 🚨
- Day 2: Try change A → C ❌ (LOCKED for 29 days)
- Day 31: Try change A → C ✅

**Scenario 3: Multiple Attempts**
- Day 0: Check eligibility → eligible = true
- Day 0: Change email A → B ✅
- Day 10: Check eligibility → eligible = false, days_remaining = 20
- Day 10: Try change B → C ❌ (20 days remaining)

---

## �📝 Notes

1. **Why send to OLD email after verification?**
   - Untuk memastikan new email valid dulu
   - Kalau langsung kirim, bisa jadi new email invalid
   - User bisa stuck tanpa akses ke akun

2. **Why 24 hours revoke expiry?**
   - Balance antara security dan user experience
   - Cukup waktu untuk user check email
   - Tidak terlalu lama untuk threat actor

3. **Why 30 days rate limit?**
   - Prevent abuse dan account takeover attempts
   - Give time untuk legitimate owner detect unauthorized changes
   - Industry standard untuk sensitive operations

4. **Why lock after revoke?**
   - Revoke indicates possible compromise
   - Extra protection layer untuk legitimate owner
   - Prevent attacker dari repeated attempts

5. **Why transaction?**
   - Atomic operation
   - Rollback jika ada error
   - Consistent state

6. **Why log to history?**
   - Audit trail
   - Compliance
   - Debug issues
   - Security analysis
