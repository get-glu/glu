import { SerializedError } from '@reduxjs/toolkit';
import { FetchBaseQueryError } from '@reduxjs/toolkit/query';
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
    'bg-red-200 text-red-900 hover:bg-red-300 dark:bg-red-600 dark:hover:bg-red-500 dark:text-red-100',
    'bg-blue-200 text-blue-900 hover:bg-blue-300 dark:bg-blue-600 dark:hover:bg-blue-500 dark:text-blue-100',
    'bg-green-200 text-green-900 hover:bg-green-300 dark:bg-green-600 dark:hover:bg-green-500 dark:text-green-100',
    'bg-yellow-200 text-yellow-900 hover:bg-yellow-300 dark:bg-yellow-600 dark:hover:bg-yellow-500 dark:text-yellow-100',
    'bg-purple-200 text-purple-900 hover:bg-purple-300 dark:bg-purple-600 dark:hover:bg-purple-500 dark:text-purple-100',
    'bg-pink-200 text-pink-900 hover:bg-pink-300 dark:bg-pink-600 dark:hover:bg-pink-500 dark:text-pink-100',
    'bg-indigo-200 text-indigo-900 hover:bg-indigo-300 dark:bg-indigo-600 dark:hover:bg-indigo-500 dark:text-indigo-100',
    'bg-orange-200 text-orange-900 hover:bg-orange-300 dark:bg-orange-600 dark:hover:bg-orange-500 dark:text-orange-100',
    'bg-gray-200 text-gray-900 hover:bg-gray-300 dark:bg-gray-600 dark:hover:bg-gray-500 dark:text-gray-100'
  ];

  return colors[Math.abs(hash) % colors.length];
}

export function getErrorMessage(error: SerializedError | FetchBaseQueryError): string {
  if ('data' in error) {
    return error.data as string;
  }
  if ('message' in error) {
    return error.message as string;
  }
  return 'An unknown error occurred';
}
