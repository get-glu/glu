import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function getLabelColor(key: string, value: string): string {
  const hash = `${key}:${value}`.split('').reduce((acc, char) => {
    return char.charCodeAt(0) + ((acc << 5) - acc);
  }, 0);

  const colors = [
    'bg-red-100 text-red-800 hover:bg-red-200 dark:bg-red-900 dark:hover:bg-red-800 dark:text-red-200',
    'bg-blue-100 text-blue-800 hover:bg-blue-200 dark:bg-blue-900 dark:hover:bg-blue-800 dark:text-blue-200',
    'bg-green-100 text-green-800 hover:bg-green-200 dark:bg-green-900 dark:hover:bg-green-800 dark:text-green-200',
    'bg-yellow-100 text-yellow-800 hover:bg-yellow-200 dark:bg-yellow-900 dark:hover:bg-yellow-800 dark:text-yellow-200',
    'bg-purple-100 text-purple-800 hover:bg-purple-200 dark:bg-purple-900 dark:hover:bg-purple-800 dark:text-purple-200',
    'bg-pink-100 text-pink-800 hover:bg-pink-200 dark:bg-pink-900 dark:hover:bg-pink-800 dark:text-pink-200',
    'bg-indigo-100 text-indigo-800 hover:bg-indigo-200 dark:bg-indigo-900 dark:hover:bg-indigo-800 dark:text-indigo-200',
    'bg-orange-100 text-orange-800 hover:bg-orange-200 dark:bg-orange-900 dark:hover:bg-orange-800 dark:text-orange-200',
    'bg-gray-100 text-gray-800 hover:bg-gray-200 dark:bg-gray-900 dark:hover:bg-gray-800 dark:text-gray-200'
  ];

  return colors[Math.abs(hash) % colors.length];
}
