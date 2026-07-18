"use client";

import { useEffect, useRef } from "react";
import { track } from "@/lib/track";

// Calendly inline widget. widget.js scans for .calendly-inline-widget on load;
// if it's already loaded (client-side nav back to this page), re-init manually.
const SCRIPT_SRC = "https://assets.calendly.com/assets/external/widget.js";

declare global {
  interface Window {
    Calendly?: {
      initInlineWidget: (opts: { url: string; parentElement: HTMLElement }) => void;
    };
  }
}

export default function CalendlyEmbed({ url }: { url: string }) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    // Calendly posts a message when a booking is completed — track it as a conversion.
    const onMessage = (e: MessageEvent) => {
      if ((e.data as { event?: string })?.event === "calendly.event_scheduled") {
        track("booking_scheduled");
      }
    };
    window.addEventListener("message", onMessage);

    if (window.Calendly && ref.current && ref.current.childElementCount === 0) {
      window.Calendly.initInlineWidget({ url, parentElement: ref.current });
    } else if (!document.querySelector(`script[src="${SCRIPT_SRC}"]`)) {
      const s = document.createElement("script");
      s.src = SCRIPT_SRC;
      s.async = true;
      document.body.appendChild(s);
    }

    return () => window.removeEventListener("message", onMessage);
  }, [url]);

  return (
    <div
      ref={ref}
      className="calendly-inline-widget"
      data-url={url}
      style={{ minWidth: "320px", height: "620px" }}
    />
  );
}
