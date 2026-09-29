---
spec: calendar-dates-tolerate-utc-stamping
category: fixed
breaking: false
---

A spec created and closed on the same local evening west of UTC passes the lifecycle gate again. `pose new-spec` stamps `created_at` with the UTC date, so at 23:30 in UTC-3 it recorded tomorrow, and the local `completed_at` a person writes by hand was refused as earlier. Between two bare dates, one day of difference is now accepted as that skew; two days, or any regression between full timestamps, is still an error. Knowledge review dates follow the same rule.
