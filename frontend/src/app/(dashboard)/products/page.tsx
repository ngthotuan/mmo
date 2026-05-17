"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  ShoppingBag,
  RefreshCw,
  Trash2,
  Plus,
  ExternalLink,
} from "lucide-react";
import { toast } from "sonner";
import { Header } from "@/components/layout/Header";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { productsApi, type CreateProductBody, type SyncProductsBody } from "@/lib/api/products";
import { channelsApi } from "@/lib/api/channels";
import type { Product } from "@/lib/types/api.types";

const PLATFORM_FILTER = [
  { value: "", label: "All" },
  { value: "tiktok", label: "TikTok" },
  { value: "facebook", label: "Facebook" },
];

function ProductCard({ product, onDelete }: { product: Product; onDelete: () => void }) {
  return (
    <div className="rounded-lg border bg-card p-4 flex flex-col gap-3">
      <div className="flex items-start gap-3">
        {product.cover_image_url ? (
          <img
            src={product.cover_image_url}
            alt={product.name}
            className="h-16 w-16 rounded-md object-cover shrink-0"
          />
        ) : (
          <div className="h-16 w-16 rounded-md bg-muted flex items-center justify-center shrink-0">
            <ShoppingBag className="h-6 w-6 text-muted-foreground" />
          </div>
        )}
        <div className="flex-1 min-w-0">
          <p className="text-sm font-medium line-clamp-2">{product.name}</p>
          <div className="flex items-center gap-2 mt-1">
            <Badge variant="outline" className="text-xs capitalize">
              {product.platform}
            </Badge>
            {product.price > 0 && (
              <span className="text-xs text-muted-foreground">
                {product.price.toLocaleString()} {product.currency}
              </span>
            )}
          </div>
        </div>
      </div>

      {product.description && (
        <p className="text-xs text-muted-foreground line-clamp-2">{product.description}</p>
      )}

      <div className="flex items-center gap-2 mt-auto">
        {product.product_url && (
          <a
            href={product.product_url}
            target="_blank"
            rel="noopener noreferrer"
            className="text-xs text-primary hover:underline flex items-center gap-1"
          >
            <ExternalLink className="h-3 w-3" />
            View
          </a>
        )}
        <Button
          size="sm"
          variant="ghost"
          className="ml-auto text-destructive hover:text-destructive"
          onClick={onDelete}
        >
          <Trash2 className="h-3.5 w-3.5" />
        </Button>
      </div>
    </div>
  );
}

function AddProductDialog({
  open,
  onClose,
}: {
  open: boolean;
  onClose: () => void;
}) {
  const qc = useQueryClient();
  const [form, setForm] = useState<CreateProductBody>({
    platform: "tiktok",
    platform_product_id: "",
    name: "",
    description: "",
    price: 0,
    currency: "USD",
    cover_image_url: "",
    product_url: "",
  });

  const { mutate, isPending } = useMutation({
    mutationFn: () => productsApi.create(form),
    onSuccess: () => {
      toast.success("Product added");
      qc.invalidateQueries({ queryKey: ["products"] });
      onClose();
    },
    onError: () => toast.error("Failed to add product"),
  });

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Add Product</DialogTitle>
        </DialogHeader>
        <div className="flex flex-col gap-3">
          <div className="grid grid-cols-2 gap-3">
            <div className="flex flex-col gap-1.5">
              <Label className="text-xs">Platform</Label>
              <select
                className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm"
                value={form.platform}
                onChange={(e) => setForm({ ...form, platform: e.target.value })}
              >
                <option value="tiktok">TikTok</option>
                <option value="facebook">Facebook</option>
              </select>
            </div>
            <div className="flex flex-col gap-1.5">
              <Label className="text-xs">Product ID</Label>
              <Input
                placeholder="Platform product ID"
                value={form.platform_product_id}
                onChange={(e) => setForm({ ...form, platform_product_id: e.target.value })}
              />
            </div>
          </div>
          <div className="flex flex-col gap-1.5">
            <Label className="text-xs">Name *</Label>
            <Input
              placeholder="Product name"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label className="text-xs">Description</Label>
            <Input
              placeholder="Short description"
              value={form.description}
              onChange={(e) => setForm({ ...form, description: e.target.value })}
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="flex flex-col gap-1.5">
              <Label className="text-xs">Price</Label>
              <Input
                type="number"
                min={0}
                step={0.01}
                value={form.price}
                onChange={(e) => setForm({ ...form, price: parseFloat(e.target.value) || 0 })}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label className="text-xs">Currency</Label>
              <Input
                placeholder="USD"
                value={form.currency}
                onChange={(e) => setForm({ ...form, currency: e.target.value })}
              />
            </div>
          </div>
          <div className="flex flex-col gap-1.5">
            <Label className="text-xs">Product URL</Label>
            <Input
              placeholder="https://..."
              value={form.product_url}
              onChange={(e) => setForm({ ...form, product_url: e.target.value })}
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label className="text-xs">Cover Image URL</Label>
            <Input
              placeholder="https://..."
              value={form.cover_image_url}
              onChange={(e) => setForm({ ...form, cover_image_url: e.target.value })}
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Cancel</Button>
          <Button
            onClick={() => mutate()}
            disabled={isPending || !form.name}
          >
            {isPending ? "Adding…" : "Add Product"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function SyncDialog({
  open,
  onClose,
}: {
  open: boolean;
  onClose: () => void;
}) {
  const qc = useQueryClient();
  const { data: channels } = useQuery({
    queryKey: ["channels"],
    queryFn: channelsApi.list,
  });
  const [form, setForm] = useState<SyncProductsBody>({
    platform: "tiktok",
    channel_id: "",
    catalog_id: "",
  });

  const { mutate, isPending } = useMutation({
    mutationFn: () => productsApi.sync(form),
    onSuccess: (res) => {
      toast.success(`Synced ${res.synced} products`);
      qc.invalidateQueries({ queryKey: ["products"] });
      onClose();
    },
    onError: (e: Error) => toast.error(e.message || "Sync failed"),
  });

  const filteredChannels = (channels ?? []).filter(
    (ch) => ch.platform === form.platform
  );

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>Sync Products from Shop</DialogTitle>
        </DialogHeader>
        <div className="flex flex-col gap-3">
          <div className="flex flex-col gap-1.5">
            <Label className="text-xs">Platform</Label>
            <select
              className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm"
              value={form.platform}
              onChange={(e) =>
                setForm({ ...form, platform: e.target.value as "tiktok" | "facebook", channel_id: "" })
              }
            >
              <option value="tiktok">TikTok Shop</option>
              <option value="facebook">Facebook Catalog</option>
            </select>
          </div>
          <div className="flex flex-col gap-1.5">
            <Label className="text-xs">Channel</Label>
            <select
              className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm"
              value={form.channel_id}
              onChange={(e) => setForm({ ...form, channel_id: e.target.value })}
            >
              <option value="">— Select channel —</option>
              {filteredChannels.map((ch) => (
                <option key={ch.id} value={ch.id}>
                  {ch.display_name || ch.username}
                </option>
              ))}
            </select>
          </div>
          {form.platform === "facebook" && (
            <div className="flex flex-col gap-1.5">
              <Label className="text-xs">Catalog ID</Label>
              <Input
                placeholder="Facebook Product Catalog ID"
                value={form.catalog_id}
                onChange={(e) => setForm({ ...form, catalog_id: e.target.value })}
              />
            </div>
          )}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Cancel</Button>
          <Button
            onClick={() => mutate()}
            disabled={isPending || !form.channel_id}
          >
            {isPending ? (
              <>
                <RefreshCw className="h-3.5 w-3.5 mr-1.5 animate-spin" />
                Syncing…
              </>
            ) : (
              <>
                <RefreshCw className="h-3.5 w-3.5 mr-1.5" />
                Sync
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export default function ProductsPage() {
  const qc = useQueryClient();
  const [platformFilter, setPlatformFilter] = useState("");
  const [addOpen, setAddOpen] = useState(false);
  const [syncOpen, setSyncOpen] = useState(false);

  const { data, isLoading } = useQuery({
    queryKey: ["products", platformFilter],
    queryFn: () => productsApi.list({ platform: platformFilter || undefined }),
  });

  const deleteMut = useMutation({
    mutationFn: productsApi.delete,
    onSuccess: () => {
      toast.success("Product deleted");
      qc.invalidateQueries({ queryKey: ["products"] });
    },
    onError: () => toast.error("Failed to delete"),
  });

  const products = data?.data ?? [];
  const total = data?.total ?? 0;

  return (
    <div className="flex flex-col gap-6 p-6">
      <Header title="Product Catalog" />

      <div className="flex items-center justify-between flex-wrap gap-3">
        <div className="flex gap-2 flex-wrap">
          {PLATFORM_FILTER.map(({ value, label }) => (
            <Button
              key={value}
              size="sm"
              variant={platformFilter === value ? "default" : "outline"}
              onClick={() => setPlatformFilter(value)}
            >
              {label}
            </Button>
          ))}
          <span className="text-sm text-muted-foreground self-center ml-2">
            {total} products
          </span>
        </div>
        <div className="flex gap-2">
          <Button size="sm" variant="outline" onClick={() => setSyncOpen(true)}>
            <RefreshCw className="h-3.5 w-3.5 mr-1.5" />
            Sync from Shop
          </Button>
          <Button size="sm" onClick={() => setAddOpen(true)}>
            <Plus className="h-3.5 w-3.5 mr-1.5" />
            Add Product
          </Button>
        </div>
      </div>

      {isLoading ? (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {Array.from({ length: 8 }).map((_, i) => (
            <Skeleton key={i} className="h-44 rounded-lg" />
          ))}
        </div>
      ) : products.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-24 text-muted-foreground gap-3">
          <ShoppingBag className="h-12 w-12 opacity-30" />
          <p className="text-sm">No products yet. Add manually or sync from TikTok Shop / Facebook Catalog.</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {products.map((p) => (
            <ProductCard
              key={p.id}
              product={p}
              onDelete={() => deleteMut.mutate(p.id)}
            />
          ))}
        </div>
      )}

      <AddProductDialog open={addOpen} onClose={() => setAddOpen(false)} />
      <SyncDialog open={syncOpen} onClose={() => setSyncOpen(false)} />
    </div>
  );
}
