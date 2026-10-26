# Webhooks and multiple local services

Burrow forwards a public HTTPS address to a service running on your machine.
The examples below use synthetic data and the current `burrowctl tunnel`, `list`,
`logs` and `close` commands. See the [client reference](client.md) for installation,
platform downloads, configuration and the owner-credential model.

## Before you share

A tunnel URL makes the application's reachable routes public. Burrow's owner
credential protects tunnel management and request logs, not the application you
expose. Use application authentication and provider signature verification where
appropriate. Never expose a database, an admin panel, real customer information
or a development server containing secrets just to try a tunnel.

## Receive a test webhook

You need a local HTTP service. For a disposable example, Node's built-in HTTP
module can acknowledge only `POST /webhook`. No package installation is needed
for this example service:

```sh
node --input-type=module - <<'JS'
import { createServer } from 'node:http';
createServer((request, response) => {
  const accepted = request.method === 'POST' && request.url === '/webhook';
  request.resume();
  response.writeHead(accepted ? 204 : 404);
  response.end();
}).listen(3000, '127.0.0.1', () => console.log('Test endpoint on port 3000'));
JS
```

In a second terminal, verify the service before adding a tunnel:

```sh
curl -i -X POST http://127.0.0.1:3000/webhook
```

Expected: `204 No Content`. This demonstrates an acknowledgement, not provider
signature validation or durable event processing. Stop if a different process is
already using the port; choose a free port consistently in both commands instead.

Then run:

```sh
burrowctl tunnel 3000
```

The interactive manager shows the assigned public URL. Append `/webhook` to that
actual URL when configuring a test provider. You can also use curl, replacing the
entire example hostname below with the URL shown by your own tunnel:

```sh
curl -i -X POST https://YOUR-ASSIGNED-HOST/webhook
```

Expected: the same `204` response. A successful connection alone does not prove
your application's background work completed. Use synthetic provider test events;
real endpoints must authenticate requests and handle retries appropriately.

## Inspect only your tunnel

In another terminal on the same machine, with the same local owner configuration:

```sh
burrowctl list
burrowctl logs YOUR-TUNNEL-ID --tail 20
burrowctl logs YOUR-TUNNEL-ID --follow
```

Replace the ID with one returned by `list`. Stop following logs with Ctrl+C.
Different servers have different owner credentials. If you used `--server`, use
the same server for list, logs and close. A public URL alone does not grant these
management permissions; never send the owner credential to a webhook provider.

## Share two local services

Start your own test services on ports 3000 and 8080, verify both locally, then run:

```sh
burrowctl tunnel 3000 8080
```

The manager lists a separate public URL for each port. It does not combine the
services behind one hostname or make an application's private routes safe.
Select a row to copy/open its URL; use `n` to add a port, `l` to inspect logs and
`?` for controls. Configure webhook callbacks against the appropriate service URL.

## Stop and clean up

```sh
burrowctl list
burrowctl close YOUR-TUNNEL-ID
```

Close every test tunnel you created, exit the manager, and stop your local test
services. Explicit closure revokes that tunnel ID; create a new tunnel for a fresh
URL. Exiting the CLI disconnects forwarding but is not the same as explicitly
closing the ID. Keep the private owner configuration: deleting it is not a cleanup
step and may remove your ability to manage other tunnels.

## Troubleshooting

- **Local request fails:** fix the service/port first. A tunnel cannot repair it.
- **Wrong method or path:** the sample accepts exactly `POST /webhook`; a browser
  address-bar visit uses GET and correctly returns 404.
- **Public request fails while local works:** inspect the manager's connection
  status, server address, local firewall and matching port. Keep the CLI running.
- **List/logs do not show a tunnel:** check the server and local owner configuration;
  do not share credentials or assume another owner's tunnel is accessible.
- **The webhook responds but processing fails:** inspect your application's own
  validation/retry handling; forwarding is not an application success guarantee.

For self-hosting prerequisites, TLS and Redis retention, read the
[server reference](server.md) and [deployment guide](deploy.md). For unresolved
problems, [contact Burrow](https://useburrow.app/support) without credentials or
private request payloads.
