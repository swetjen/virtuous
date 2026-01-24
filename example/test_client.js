// Basic test client for the generated JS SDK.

import { createClient } from "./client.gen.js";

async function main() {
  const client = createClient("http://localhost:8000");
  try {
    const response = await client.States.getByCode({ code: "mn" });
    console.log(response);
  } catch (err) {
    console.error("request failed:", err);
  }
}

main();
