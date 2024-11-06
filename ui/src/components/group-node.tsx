export function GroupNode({ data }: { data: any }) {
  return (
    <div className="min-h-[200px] min-w-[800px] rounded-lg p-4">
      <div className="absolute -top-3 left-2 bg-background px-2 text-sm text-muted-foreground">
        {data.label}
      </div>
    </div>
  );
}
