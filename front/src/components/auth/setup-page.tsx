'use client';

import { useState } from 'react';
import { Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { authApi } from '@/lib/api';
import { setTokens } from '@/lib/api/client';

interface SetupPageProps {
  onDone: () => void;
}

export function SetupPage({ onDone }: SetupPageProps) {
  const [form, setForm] = useState({
    username: 'admin', password: '', displayName: 'Platform Admin',
    email: '', organization: 'default', setupToken: '',
  });
  const [err, setErr] = useState('');
  const [loading, setLoading] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErr('');
    setLoading(true);
    try {
      const r = await authApi.setup(form);
      setTokens(r.tokens.accessToken, r.tokens.refreshToken);
      onDone();
    } catch (e: unknown) {
      setErr(e instanceof Error ? e.message : 'Setup failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex items-center justify-center min-h-screen bg-gradient-to-br from-background via-background to-violet-50 dark:to-violet-950/20 p-4">
      <Card className="w-full max-w-lg shadow-xl border-0 shadow-violet-500/5">
        <CardHeader className="text-center space-y-3">
          <div className="mx-auto w-12 h-12 rounded-xl bg-gradient-to-br from-violet-600 to-fuchsia-500 text-white font-bold flex items-center justify-center text-lg">SH</div>
          <CardTitle className="text-xl">First-Time Setup</CardTitle>
          <CardDescription>Create the first admin account</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={submit} className="space-y-4">
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-2">
                <label className="text-sm font-medium">Username</label>
                <Input value={form.username} onChange={(e) => setForm({ ...form, username: e.target.value })} />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">Display Name</label>
                <Input value={form.displayName} onChange={(e) => setForm({ ...form, displayName: e.target.value })} />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">Password</label>
                <Input type="password" value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">Organization</label>
                <Input value={form.organization} onChange={(e) => setForm({ ...form, organization: e.target.value })} />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">Email</label>
                <Input value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">Setup Token</label>
                <Input value={form.setupToken} onChange={(e) => setForm({ ...form, setupToken: e.target.value })} placeholder="Leave empty if not configured" />
              </div>
            </div>
            {err && <p className="text-sm text-destructive">{err}</p>}
            <Button type="submit" className="w-full bg-gradient-to-r from-violet-600 to-fuchsia-500 hover:from-violet-700 hover:to-fuchsia-600" disabled={loading}>
              {loading && <Loader2 className="h-4 w-4 mr-2 animate-spin" />}
              Initialize Platform
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
