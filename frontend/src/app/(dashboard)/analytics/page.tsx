"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { BarChart3, Eye, Heart, MessageCircle, Share2, Loader2 } from "lucide-react";
import { Header } from "@/components/layout/Header";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { analyticsApi } from "@/lib/api/analytics";

const DAY_OPTIONS = [7, 14, 30, 90];

function StatCard({ icon: Icon, label, value }: { icon: React.ElementType; label: string; value: number }) {
  return (
    <div className="rounded-lg border bg-card p-4 flex items-center gap-4">
      <div className="rounded-full bg-primary/10 p-2.5">
        <Icon className="h-5 w-5 text-primary" />
      </div>
      <div>
        <p className="text-xs text-muted-foreground">{label}</p>
        <p className="text-2xl font-bold">{value.toLocaleString()}</p>
      </div>
    </div>
  );
}

export default function AnalyticsPage() {
  const [days, setDays] = useState(30);

  const { data: overview, isLoading: overviewLoading } = useQuery({
    queryKey: ["analytics-overview", days],
    queryFn: () => analyticsApi.overview(days),
  });

  const { data: postsData, isLoading: postsLoading } = useQuery({
    queryKey: ["analytics-posts"],
    queryFn: () => analyticsApi.listPosts({ page: 1, per_page: 20 }),
  });

  const stats = overview?.data;

  return (
    <div className="flex flex-col gap-6 p-6">
      <Header title="Analytics" />

      {/* Time range filter */}
      <div className="flex gap-2">
        {DAY_OPTIONS.map((d) => (
          <Button
            key={d}
            size="sm"
            variant={days === d ? "default" : "outline"}
            onClick={() => setDays(d)}
          >
            {d}d
          </Button>
        ))}
      </div>

      {/* Overview stats */}
      {overviewLoading ? (
        <div className="grid grid-cols-2 gap-4 lg:grid-cols-5">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-20 rounded-lg" />
          ))}
        </div>
      ) : (
        <div className="grid grid-cols-2 gap-4 lg:grid-cols-5">
          <StatCard icon={BarChart3} label="Posts" value={stats?.post_count ?? 0} />
          <StatCard icon={Eye} label="Views" value={stats?.total_views ?? 0} />
          <StatCard icon={Heart} label="Likes" value={stats?.total_likes ?? 0} />
          <StatCard icon={MessageCircle} label="Comments" value={stats?.total_comments ?? 0} />
          <StatCard icon={Share2} label="Shares" value={stats?.total_shares ?? 0} />
        </div>
      )}

      {/* Posts table */}
      <div className="rounded-lg border bg-card">
        <div className="p-4 border-b">
          <h2 className="font-semibold text-sm">Post Performance</h2>
        </div>
        {postsLoading ? (
          <div className="p-4 space-y-3">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-10 rounded" />
            ))}
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Platform</TableHead>
                <TableHead>Synced</TableHead>
                <TableHead className="text-right">Views</TableHead>
                <TableHead className="text-right">Likes</TableHead>
                <TableHead className="text-right">Comments</TableHead>
                <TableHead className="text-right">Shares</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {postsData?.data.length === 0 && (
                <TableRow>
                  <TableCell colSpan={6} className="text-center text-muted-foreground py-12">
                    No analytics data yet. Analytics sync runs daily after publishing.
                  </TableCell>
                </TableRow>
              )}
              {postsData?.data.map((row) => (
                <TableRow key={row.publish_job_id}>
                  <TableCell className="capitalize font-medium">{row.platform}</TableCell>
                  <TableCell className="text-muted-foreground text-xs">
                    {new Date(row.synced_at).toLocaleDateString()}
                  </TableCell>
                  <TableCell className="text-right">{row.views.toLocaleString()}</TableCell>
                  <TableCell className="text-right">{row.likes.toLocaleString()}</TableCell>
                  <TableCell className="text-right">{row.comments.toLocaleString()}</TableCell>
                  <TableCell className="text-right">{row.shares.toLocaleString()}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </div>
    </div>
  );
}
