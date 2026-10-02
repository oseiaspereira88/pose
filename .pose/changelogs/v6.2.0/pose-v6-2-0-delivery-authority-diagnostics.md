---
spec: pose-v6-2-0-delivery-authority-diagnostics
category: fixed
breaking: false
---

Delivery graph failures are reported once with their originating spec instead
of being repeated against unrelated completed specs. Human authority issuer
pins receive format diagnostics, authority claims use their envelope schema,
and excluded lifecycle paths remain portable between checkouts.
