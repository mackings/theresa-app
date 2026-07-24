const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8090";

export class ApiError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const isFormData = init?.body instanceof FormData;

  const res = await fetch(`${API_URL}${path}`, {
    ...init,
    credentials: "include",
    headers: {
      ...(isFormData ? {} : { "Content-Type": "application/json" }),
      ...init?.headers,
    },
  });

  const body = await res.json().catch(() => ({}));

  if (!res.ok) {
    throw new ApiError(res.status, body.error ?? "something went wrong");
  }

  return body as T;
}
