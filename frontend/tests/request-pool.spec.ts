import { expect, test } from "@playwright/test";
import { RequestPool } from "../src/requestPool";

test("queued cancellation never starts stale work and frees queue capacity", async () => {
  const pool = new RequestPool(1, 1);
  const signal = new AbortController().signal;
  let release!: () => void;
  const first = pool.run(signal, () => new Promise<void>((resolve) => { release = resolve; }));
  await Promise.resolve();
  const cancelled = new AbortController();
  let ran = false;
  const stale = pool.run(cancelled.signal, async () => { ran = true; });
  await expect(pool.run(signal, async () => {})).rejects.toThrow("Too many pending requests");
  cancelled.abort();
  await expect(stale).rejects.toMatchObject({ name: "AbortError" });
  const next = pool.run(signal, async () => "next");
  release();
  await first;
  expect(await next).toBe("next");
  expect(ran).toBe(false);
});

test("failure releases the active slot and allows later requests", async () => {
  const pool = new RequestPool(1);
  const signal = new AbortController().signal;
  await expect(pool.run(signal, async () => { throw new Error("offline"); })).rejects.toThrow("offline");
  expect(await pool.run(signal, async () => "recovered")).toBe("recovered");
});
