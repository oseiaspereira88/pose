---
spec: pose-stdio-server-honours-sigterm
category: fixed
breaking: false
refs:
---

`pose serve-mcp --stdio` exits on SIGTERM, with status 0. It caught the signal
and then kept waiting on stdin, looking at it only when the next request
arrived — which it then dropped unanswered. A client or supervisor stopping the
server the usual way left it running until the next call, and that call failed
with a closed connection.
