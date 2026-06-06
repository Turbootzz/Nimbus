import Image from 'next/image'

// Brand logo. The source image is 272x202; `size` sets the rendered width and
// the height scales automatically to preserve the aspect ratio.
export default function NimbusLogo({
  size = 32,
  priority = false,
}: {
  size?: number
  priority?: boolean
}) {
  return (
    <Image
      src="/images/logo.png"
      alt="Nimbus"
      width={272}
      height={202}
      style={{ width: size, height: 'auto' }}
      priority={priority}
    />
  )
}
