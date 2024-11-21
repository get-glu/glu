export default function Root() {
  return (
    <div className="mx-auto flex items-center justify-center">
      <div className="rounded-lg border bg-card p-8 shadow-md">
        <h3 className="mb-2 text-xl font-semibold">Welcome to Glu</h3>
        <p className="text-muted-foreground">Select a pipeline from the sidebar to get started.</p>
      </div>
    </div>
  );
}
