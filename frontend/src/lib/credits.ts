import { apiFetch } from "@/lib/api";

export interface CreditBalance {
  balance_kobo: number;
  free_trial_seconds_remaining: number;
  percent_remaining: number;
}

export function getCreditBalance(): Promise<CreditBalance> {
  return apiFetch<CreditBalance>("/api/payments/balance");
}

export function initiatePurchase(amountNaira: number): Promise<{ payment_link: string }> {
  return apiFetch<{ payment_link: string }>("/api/payments/initiate", {
    method: "POST",
    body: JSON.stringify({ amount_naira: amountNaira }),
  });
}

export function nairaFromKobo(kobo: number): number {
  return kobo / 100;
}

// A narrow no-break space after the Naira sign keeps its two horizontal
// strokes (that's the actual glyph design for U+20A6, not a rendering bug)
// from visually bleeding into the first digit and reading like a
// strikethrough price at typical UI font weights.
export function formatNaira(kobo: number): string {
  return `₦ ${nairaFromKobo(kobo).toLocaleString("en-NG", { maximumFractionDigits: 2 })}`;
}
