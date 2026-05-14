"use client";

import { useEffect, useState } from "react";
import { LoginInput, SignupInput, User } from "@/features/auth/types/auth";
import {
  authenticateUser,
  clearCurrentSession,
  createUser,
  getCurrentUser,
  subscribeAuthStorage,
} from "@/lib/storage/authStorage";

type UseAuthResult = {
  user: User | null;
  loading: boolean;
  login: (input: LoginInput) => Promise<User>;
  signup: (input: SignupInput) => Promise<User>;
  logout: () => Promise<void>;
};

export function useAuth(): UseAuthResult {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let isMounted = true;

    const load = async () => {
      try {
        const nextUser = await getCurrentUser();
        if (isMounted) {
          setUser(nextUser);
        }
      } finally {
        if (isMounted) {
          setLoading(false);
        }
      }
    };

    void load();

    const unsubscribe = subscribeAuthStorage(() => {
      setLoading(true);
      void load();
    });

    return () => {
      isMounted = false;
      unsubscribe();
    };
  }, []);

  return {
    user,
    loading,
    login: async (input) => {
      const nextUser = await authenticateUser(input);
      setUser(nextUser);
      return nextUser;
    },
    signup: async (input) => {
      const nextUser = await createUser(input);
      setUser(nextUser);
      return nextUser;
    },
    logout: async () => {
      await clearCurrentSession();
      setUser(null);
    },
  };
}