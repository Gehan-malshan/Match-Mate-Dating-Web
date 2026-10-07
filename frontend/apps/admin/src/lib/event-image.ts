const maxBytes = 650 * 1024

export async function prepareEventImage(file: File): Promise<string> {
  if (!['image/jpeg', 'image/png', 'image/webp'].includes(file.type) || file.size > 15 * 1024 * 1024) {
    throw new Error('Choose a JPEG, PNG, or WebP image under 15 MB.')
  }
  const bitmap = await createImageBitmap(file)
  try {
    if (bitmap.width < 1 || bitmap.height < 1) throw new Error('Image dimensions are invalid.')
    const canvas = document.createElement('canvas')
    for (const edge of [1600, 1200, 900]) {
      const scale = Math.min(1, edge / Math.max(bitmap.width, bitmap.height))
      canvas.width = Math.max(1, Math.round(bitmap.width * scale))
      canvas.height = Math.max(1, Math.round(bitmap.height * scale))
      const context = canvas.getContext('2d')
      if (!context) throw new Error('This browser cannot prepare the image.')
      context.fillStyle = '#ffffff'
      context.fillRect(0, 0, canvas.width, canvas.height)
      context.drawImage(bitmap, 0, 0, canvas.width, canvas.height)
      for (const quality of [0.82, 0.68, 0.52, 0.38]) {
        const base64 = canvas.toDataURL('image/jpeg', quality).split(',')[1]
        if (base64 && Math.floor(base64.length * 3 / 4) <= maxBytes) return base64
      }
    }
    throw new Error('This image is too detailed to upload. Try a smaller image.')
  } finally { bitmap.close() }
}
