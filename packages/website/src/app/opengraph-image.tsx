import { readFile } from "node:fs/promises";
import { join } from "node:path";

export const alt =
  "Burrow. Localhost, shared. Public HTTPS tunnels. useburrow.app.";
export const size = { width: 1200, height: 628 };
export const contentType = "image/png";

export default async function OpenGraphImage() {
  const image = await readFile(
    join(process.cwd(), "public/brand-kit/images/social.png"),
  );
  return new Response(new Uint8Array(image), {
    headers: { "Content-Type": contentType },
  });
}
