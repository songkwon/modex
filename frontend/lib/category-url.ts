import type { Category } from "@/types/modex";

export function categoryRouteSegment(category: Category): string {
  return encodeURIComponent(category.name || category.key || category.id);
}

export function categoryHref(category: Category): string {
  return `/categories/${categoryRouteSegment(category)}`;
}

export function findCategoryByRouteSegment(categories: Category[], segment: string): Category | null {
  const decoded = decodeURIComponent(segment);
  for (const category of categories) {
    if (category.name === decoded || category.id === decoded || category.key === decoded) {
      return category;
    }
    const found = findCategoryByRouteSegment(category.children || [], segment);
    if (found) return found;
  }
  return null;
}
