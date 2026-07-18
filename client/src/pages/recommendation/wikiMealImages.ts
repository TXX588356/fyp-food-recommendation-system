type WikipediaSearchResponse = {
  pages?: Array<{
    title?: string
  }>
}

type WikipediaSummaryResponse = {
  originalimage?: {
    source?: string
  }
  thumbnail?: {
    source?: string
  }
}

type CachedImage = {
  imageUrl: string
  cachedAt: string
}

const CACHE_PREFIX = 'wiki-meal-image:'
const SEARCH_URL = 'https://en.wikipedia.org/w/rest.php/v1/search/page'
const SUMMARY_URL = 'https://en.wikipedia.org/api/rest_v1/page/summary'

const cacheKey = (mealName: string) => {
  return `${CACHE_PREFIX}${mealName.trim().toLowerCase()}`
}

const readCachedImage = (mealName: string) => {
  try {
    const rawValue = localStorage.getItem(cacheKey(mealName))
    if (!rawValue) {
      return undefined
    }

    const parsedValue = JSON.parse(rawValue) as CachedImage
    return typeof parsedValue.imageUrl === 'string' ? parsedValue.imageUrl : undefined
  } catch {
    localStorage.removeItem(cacheKey(mealName))
    return undefined
  }
}

const cacheImage = (mealName: string, imageUrl: string) => {
  localStorage.setItem(
    cacheKey(mealName),
    JSON.stringify({
      imageUrl,
      cachedAt: new Date().toISOString(),
    }),
  )
}

const isAllowedWikipediaImageURL = (rawURL: string) => {
  try {
    const url = new URL(rawURL)
    return url.protocol === 'https:' && url.hostname.endsWith('.wikimedia.org')
  } catch {
    return false
  }
}

export const resolveWikipediaMealImage = async (
  mealName: string,
  signal?: AbortSignal,
) => {
  const normalizedName = mealName.trim()
  if (!normalizedName) {
    return ''
  }

  const cachedImage = readCachedImage(normalizedName)
  if (cachedImage !== undefined) {
    return cachedImage
  }

  try {
    const searchParams = new URLSearchParams({
      q: `${normalizedName} food`,
      limit: '3',
    })
    const searchResponse = await fetch(`${SEARCH_URL}?${searchParams.toString()}`, { signal })
    if (!searchResponse.ok) {
      return ''
    }

    const searchPayload = (await searchResponse.json()) as WikipediaSearchResponse
    const titles = (searchPayload.pages ?? [])
      .map((page) => page.title?.trim() ?? '')
      .filter(Boolean)

    for (const title of titles) {
      const summaryResponse = await fetch(
        `${SUMMARY_URL}/${encodeURIComponent(title)}`,
        { signal },
      )
      if (!summaryResponse.ok) {
        continue
      }

      const summaryPayload = (await summaryResponse.json()) as WikipediaSummaryResponse
      const imageUrl =
        summaryPayload.thumbnail?.source?.trim() ??
        summaryPayload.originalimage?.source?.trim() ??
        ''

      if (imageUrl && isAllowedWikipediaImageURL(imageUrl)) {
        cacheImage(normalizedName, imageUrl)
        return imageUrl
      }
    }

    cacheImage(normalizedName, '')
    return ''
  } catch (error) {
    if (error instanceof DOMException && error.name === 'AbortError') {
      return ''
    }

    return ''
  }
}
