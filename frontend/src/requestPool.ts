export class RequestPoolOverloadError extends Error {
  constructor() {
    super("request pool overloaded");
    this.name = "RequestPoolOverloadError";
  }
}

// Queries deduplicate through React Query. This pool bounds transport work,
// including manual actions, and removes cancelled requests before dispatch.
export class RequestPool {
  private active = 0;
  private waiting: Array<() => void> = [];

  constructor(private readonly limit = 6, private readonly queueLimit = 64) {}

  async run<T>(signal: AbortSignal, work: () => Promise<T>): Promise<T> {
    signal.throwIfAborted();
    await new Promise<void>((resolve, reject) => {
      const abort = () => {
        this.waiting = this.waiting.filter((entry) => entry !== start);
        reject(signal.reason);
      };
      const start = () => {
        signal.removeEventListener("abort", abort);
        this.active++;
        resolve();
      };
      if (this.active < this.limit) start();
      else if (this.waiting.length >= this.queueLimit)
        reject(new RequestPoolOverloadError());
      else {
        this.waiting.push(start);
        signal.addEventListener("abort", abort, { once: true });
      }
    });
    try {
      signal.throwIfAborted();
      return await work();
    } finally {
      this.active--;
      this.waiting.shift()?.();
    }
  }
}

export const requestPool = new RequestPool();
