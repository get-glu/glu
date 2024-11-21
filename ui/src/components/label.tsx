import { getLabelColor } from '@/lib/utils';
import { Badge } from './ui/badge';

interface LabelProps {
  labelKey: string;
  value: string;
}

export function Label({ labelKey, value }: LabelProps) {
  return (
    <Badge
      key={`${labelKey}-${value}`}
      className={`whitespace-nowrap text-xs font-light ${getLabelColor(labelKey, value)}`}
    >
      {labelKey}:{' '}
      {value.match('^https?://.*') ? (
        <a className="ml-1 hover:underline" href={value} target="_blank">
          {value}
        </a>
      ) : (
        value
      )}
    </Badge>
  );
}
