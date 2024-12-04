import { cn, getLabelColor } from '@/lib/utils';
import { Badge } from './ui/badge';

interface LabelProps {
  labelKey: string;
  value: string;
  className?: string;
}

export function Label({ labelKey, value, className }: LabelProps) {
  return (
    <Badge
      key={`${labelKey}-${value}`}
      className={cn(
        `whitespace-nowrap rounded-md text-xs font-light ${getLabelColor(labelKey, value)}`,
        className
      )}
    >
      {labelKey}:{' '}
      {value.match('^https?://.*') ? (
        <a className="ml-1 hover:underline" href={value} target="_blank" rel="noreferrer">
          {value}
        </a>
      ) : (
        value
      )}
    </Badge>
  );
}
