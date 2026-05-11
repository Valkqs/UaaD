import { useEffect, useState } from 'react';
import { getUnreadNotificationCount } from '../api/endpoints';
import { useAuth } from '../context/AuthContext';
import { subscribeNotificationState } from '../utils/notificationState';

export function useNotificationCount() {
  const { isAuthenticated, session } = useAuth();
  const [state, setState] = useState(() => ({
    count: 0,
    isLoading: false,
  }));

  useEffect(() => {
    let active = true;

    if (!isAuthenticated) {
      return undefined;
    }

    const load = () =>
      getUnreadNotificationCount()
        .then((count) => {
          if (active) {
            setState({
              count,
              isLoading: false,
            });
          }
        })
        .catch(() => {
          if (active) {
            setState({
              count: 0,
              isLoading: false,
            });
          }
        });

    const refresh = () => {
      void load();
    };

    refresh();

    const unsubscribe = subscribeNotificationState(session?.userId, refresh);
    const refreshOnVisible = () => {
      if (document.visibilityState === 'visible') {
        refresh();
      }
    };

    window.addEventListener('focus', refresh);
    document.addEventListener('visibilitychange', refreshOnVisible);

    return () => {
      active = false;
      unsubscribe();
      window.removeEventListener('focus', refresh);
      document.removeEventListener('visibilitychange', refreshOnVisible);
    };
  }, [isAuthenticated, session?.userId]);

  return {
    count: isAuthenticated ? state.count : 0,
    isLoading: isAuthenticated ? state.isLoading : false,
  };
}
