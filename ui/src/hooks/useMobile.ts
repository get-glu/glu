import { useMediaQuery } from '@uidotdev/usehooks';

/**
 * Hook that returns true if the current viewport is considered mobile-sized (< 768px)
 * @returns boolean indicating if the device is mobile-sized
 */
export const useIsMobile = (): boolean => {
  return useMediaQuery('only screen and (max-width: 768px)');
};
