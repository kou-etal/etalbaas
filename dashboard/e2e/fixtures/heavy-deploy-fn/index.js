const http = require("http");

http.createServer((req, res) => {
  let body = "";
  req.on("data", (c) => (body += c));
  req.on("end", () => {
    res.writeHead(200, { "Content-Type": "application/json" });
    res.end(
      JSON.stringify({
        kind: "heavy-deploy",
        echo: JSON.parse(body || "{}"),
        ts: Date.now(),
      }),
    );
  });
}).listen(8080);
