/**
 * Crypto Utility - Secure API Key Storage
 * 
 * Encrypts sensitive data before storing in localStorage using Web Crypto API.
 * Uses AES-GCM encryption with a key derived from browser fingerprint + salt.
 */

// Constants
const ALGORITHM = 'AES-GCM'
const KEY_LENGTH = 256
const IV_LENGTH = 12 // 96 bits for GCM
const SALT_KEY = 'crypto_salt_v1'

/**
 * Generate a consistent encryption key from browser fingerprint
 */
async function deriveEncryptionKey(): Promise<CryptoKey> {
    // Get or create salt
    if (typeof window === "undefined" || typeof localStorage === "undefined" || typeof crypto === "undefined") {
        return {} as CryptoKey // Should not be called during build
    }
    let salt = localStorage.getItem(SALT_KEY)
    if (!salt) {
        const saltBuffer = crypto.getRandomValues(new Uint8Array(16))
        salt = Array.from(saltBuffer).map(b => b.toString(16).padStart(2, '0')).join('')
        localStorage.setItem(SALT_KEY, salt)
    }

    // Create a simple fingerprint from browser characteristics
    const fingerprint = [
        typeof navigator !== "undefined" ? navigator.userAgent : 'build-agent',
        typeof navigator !== "undefined" ? navigator.language : 'en-US',
        typeof window !== "undefined" ? new Date().getTimezoneOffset().toString() : '0',
        typeof window !== "undefined" ? screen.colorDepth.toString() : '24',
        salt || 'static-salt'
    ].join('|')

    // Convert fingerprint to key material
    const encoder = new TextEncoder()
    const keyMaterial = await crypto.subtle.importKey(
        'raw',
        encoder.encode(fingerprint),
        'PBKDF2',
        false,
        ['deriveBits', 'deriveKey']
    )

    // Derive actual encryption key
    const saltBytes = new Uint8Array(salt.match(/.{2}/g)!.map(byte => parseInt(byte, 16)))
    
    return crypto.subtle.deriveKey(
        {
            name: 'PBKDF2',
            salt: saltBytes,
            iterations: 100000,
            hash: 'SHA-256'
        },
        keyMaterial,
        { name: ALGORITHM, length: KEY_LENGTH },
        false,
        ['encrypt', 'decrypt']
    )
}

/**
 * Encrypt a string value
 */
export async function encryptValue(plaintext: string): Promise<string> {
    if (typeof window === "undefined" || typeof crypto === "undefined" || typeof crypto.subtle === "undefined") {
        return ""
    }
    try {
        const key = await deriveEncryptionKey()
        const encoder = new TextEncoder()
        const data = encoder.encode(plaintext)

        // Generate random IV
        const iv = crypto.getRandomValues(new Uint8Array(IV_LENGTH))

        // Encrypt
        const encrypted = await crypto.subtle.encrypt(
            { name: ALGORITHM, iv },
            key,
            data
        )

        // Combine IV + ciphertext and encode as base64
        const combined = new Uint8Array(iv.length + encrypted.byteLength)
        combined.set(iv)
        combined.set(new Uint8Array(encrypted), iv.length)

        return btoa(String.fromCharCode(...Array.from(combined)))
    } catch (error) {
        console.error('CRYPTO_ENCRYPT_ERROR details:', error)
        console.trace('CRYPTO_ENCRYPT call stack:')
        throw new Error('CRYPTO_ENCRYPT_FAIL')
    }
}

/**
 * Decrypt an encrypted string
 */
export async function decryptValue(encrypted: string): Promise<string> {
    if (typeof window === "undefined" || typeof crypto === "undefined" || typeof crypto.subtle === "undefined") {
        return ""
    }
    try {
        const key = await deriveEncryptionKey()

        // Decode from base64
        const combined = Uint8Array.from(atob(encrypted), c => c.charCodeAt(0))

        // Extract IV and ciphertext
        const iv = combined.slice(0, IV_LENGTH)
        const ciphertext = combined.slice(IV_LENGTH)

        // Decrypt
        const decrypted = await crypto.subtle.decrypt(
            { name: ALGORITHM, iv },
            key,
            ciphertext
        )

        // Decode to string
        const decoder = new TextDecoder()
        return decoder.decode(decrypted)
    } catch (error) {
        console.error('CRYPTO_DECRYPT_ERROR details:', error)
        throw new Error('CRYPTO_DECRYPT_FAIL')
    }
}

/**
 * Securely store a value in localStorage (encrypted)
 */
export async function secureStore(key: string, value: string): Promise<void> {
    const encrypted = await encryptValue(value)
    localStorage.setItem(`enc_${key}`, encrypted)
}

/**
 * Securely retrieve a value from localStorage (decrypt)
 */
export async function secureRetrieve(key: string): Promise<string | null> {
    const encrypted = localStorage.getItem(`enc_${key}`)
    if (!encrypted) return null

    try {
        return await decryptValue(encrypted)
    } catch (error) {
        console.error(`Failed to decrypt key: ${key}`, error)
        return null
    }
}

/**
 * Remove a secure value from localStorage
 */
export function secureRemove(key: string): void {
    localStorage.removeItem(`enc_${key}`)
}

/**
 * Check if a secure value exists
 */
export function secureHas(key: string): boolean {
    return localStorage.getItem(`enc_${key}`) !== null
}

/**
 * Migrate plaintext localStorage value to encrypted
 */
export async function migrateToSecure(key: string): Promise<boolean> {
    try {
        const plainValue = localStorage.getItem(key)
        if (!plainValue) return false

        await secureStore(key, plainValue)
        localStorage.removeItem(key) // Remove plaintext version
        return true
    } catch (error) {
        console.error('Migration error:', error)
        return false
    }
}
