import Link from "next/link";
import { Button } from "@/components/ui/button";

export default function NotFound() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center gap-4 px-6 text-center">
      <p className="font-mono text-sm text-muted-foreground">404</p>
      <h1 className="text-2xl font-semibold tracking-tight">Page not found</h1>
      <p className="max-w-prose text-sm text-muted-foreground">
        The page you are looking for does not exist, or you may not have access
        to it.
      </p>
      <Button className="mt-2" render={<Link href="/" />}>
        Go home
      </Button>
    </main>
  );
}
