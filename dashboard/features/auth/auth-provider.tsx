"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react";
import type { Session, User } from "@supabase/gotrue-js";
import { useRouter } from "next/navigation";
import { useQueryClient } from "@tanstack/react-query";
import { gotrue } from "@/lib/auth/gotrue-client";

interface AuthContextValue {
  user: User | null;
  isLoading: boolean;
  signInWithProvider: (provider: "github" | "google") => Promise<void>;
  signOut: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

function setAuthCookie(session: Session | null) {
  if (session?.access_token) {
    document.cookie = `etalbaas-auth-token=${session.access_token}; path=/; max-age=${60 * 60 * 24 * 7}; SameSite=Lax; Secure`;
  } else {
    document.cookie = "etalbaas-auth-token=; path=/; max-age=0";
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const router = useRouter();
  const queryClient = useQueryClient();

  useEffect(() => {
    gotrue
      .getSession()
      .then(({ data }) => {
        setUser(data.session?.user ?? null);
        setAuthCookie(data.session);
      })
      .catch(() => {
        setUser(null);
        setAuthCookie(null);
      })
      .finally(() => setIsLoading(false));

    const {
      data: { subscription },
    } = gotrue.onAuthStateChange((_event, session) => {
      setUser(session?.user ?? null);
      setAuthCookie(session);
      setIsLoading(false);
    });

    return () => subscription.unsubscribe();
  }, []);

  const signInWithProvider = useCallback(
    async (provider: "github" | "google") => {
      const redirectTo = `${window.location.origin}/auth/callback`;
      try {
        await gotrue.signInWithOAuth({
          provider,
          options: { redirectTo },
        });
      } catch (err) {
        console.error("OAuth sign-in failed:", err);
        throw err;
      }
    },
    []
  );

  const signOut = useCallback(async () => {
    await gotrue.signOut();
    setUser(null);
    setAuthCookie(null);
    queryClient.clear();
    router.push("/login");
  }, [queryClient, router]);

  return (
    <AuthContext.Provider value={{ user, isLoading, signInWithProvider, signOut }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
