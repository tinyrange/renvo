const RETRYABLE_STATUS = new Set([408, 425, 429, 500, 502, 503, 504]);

function delay(milliseconds) {
  return new Promise((resolve) => setTimeout(resolve, milliseconds));
}

// fetchAsset retries short-lived server and proxy failures while preserving
// ordinary HTTP errors for the caller to report with its asset context.
export async function fetchAsset(url, options = {}) {
  const attempts = Math.max(1, options.attempts || 3);
  const fetcher = options.fetcher || globalThis.fetch;
  const pause = options.pause || delay;
  let lastError;
  for (let attempt = 0; attempt < attempts; attempt++) {
    try {
      const response = await fetcher(url);
      if (response.ok || !RETRYABLE_STATUS.has(response.status) || attempt + 1 === attempts) return response;
    } catch (error) {
      lastError = error;
      if (attempt + 1 === attempts) throw error;
    }
    await pause(100 * (1 << attempt));
  }
  throw lastError;
}
