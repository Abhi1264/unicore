import Link from "next/link";
import { Button } from "@/components/ui/button";

export const metadata = {
  title: "Offline",
};

export default function OfflinePage() {
  return (
    <div className="flex min-h-full flex-1 flex-col items-center justify-center unicore-grid px-6 text-center">
      <p className="text-3xl font-semibold tracking-tight text-ink">Unicore</p>
      <h1 className="mt-4 text-xl font-medium text-ink">You are offline</h1>
      <p className="mt-2 max-w-sm text-sm text-muted-foreground">
        The shell is available offline. Reconnect to load live campus data.
      </p>
      <Button className="mt-8" render={<Link href="/" />}>
        Try again
      </Button>
    </div>
  );
}
