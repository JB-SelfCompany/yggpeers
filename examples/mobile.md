# Mobile Usage Examples

## Android (Kotlin)

### Basic Usage

```kotlin
import yggpeers.Mobile

class YggdrasilPeerManager {
    private val manager = Mobile.NewManager()

    init {
        // Set log callback
        manager.setLogCallback(object : Mobile.LogCallback {
            override fun onLog(level: String, message: String) {
                Log.d("YggPeers", "[$level] $message")
            }
        })
    }

    suspend fun getBestPeers(): List<Peer> {
        return withContext(Dispatchers.IO) {
            try {
                val peersJSON = manager.getBestPeers(10, "tls,quic")
                val jsonArray = JSONArray(peersJSON)

                (0 until jsonArray.length()).map { i ->
                    val peer = jsonArray.getJSONObject(i)
                    Peer(
                        address = peer.getString("Address"),
                        protocol = peer.getString("Protocol"),
                        region = peer.getString("Region"),
                        rtt = peer.optLong("RTT", 0)
                    )
                }
            } catch (e: Exception) {
                Log.e("YggPeers", "Failed to get peers", e)
                emptyList()
            }
        }
    }

    fun checkPeersAsync(peers: List<Peer>, callback: (Int, Int) -> Unit) {
        val peersJSON = JSONArray(peers.map { it.toJSON() }).toString()

        manager.checkPeersAsync(peersJSON, object : Mobile.CheckCallback {
            override fun onPeerChecked(address: String, available: Boolean, rtt: Long) {
                Log.d("YggPeers", "$address: ${if (available) "✓" else "✗"} (${rtt}ms)")
            }

            override fun onCheckComplete(available: Int, total: Int) {
                callback(available, total)
            }
        })
    }
}

data class Peer(
    val address: String,
    val protocol: String,
    val region: String,
    val rtt: Long
)
```

### In Activity/Fragment

```kotlin
class MainActivity : AppCompatActivity() {
    private lateinit var peerManager: YggdrasilPeerManager

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)

        peerManager = YggdrasilPeerManager()

        lifecycleScope.launch {
            val peers = peerManager.getBestPeers()
            updateUI(peers)
        }
    }

    private fun updateUI(peers: List<Peer>) {
        // Update RecyclerView or list
        peersAdapter.submitList(peers)
    }
}
```

### Filtering Peers

```kotlin
suspend fun getGermanTLSPeers(): String {
    return withContext(Dispatchers.IO) {
        manager.getAvailablePeers(
            protocol = "tls",
            region = "germany",
            maxRTTMs = 200  // 200ms max
        )
    }
}
```

### Cache Management

```kotlin
// Refresh cache manually
lifecycleScope.launch {
    withContext(Dispatchers.IO) {
        manager.refreshCache()
    }
}

// Clear cache
manager.clearCache()
```

## iOS (Swift)

### Basic Usage

```swift
import Yggpeers

class YggdrasilPeerManager {
    private let manager = MobileNewManager()

    init() {
        // Set log callback
        manager?.setLogCallback(LogCallbackImpl())
    }

    func getBestPeers() async throws -> [Peer] {
        let peersJSON = try manager?.getBestPeers(10, protocol: "tls,quic")
        guard let jsonData = peersJSON?.data(using: .utf8) else {
            return []
        }

        let jsonArray = try JSONSerialization.jsonObject(with: jsonData) as? [[String: Any]]
        return jsonArray?.compactMap { Peer(from: $0) } ?? []
    }
}

class LogCallbackImpl: NSObject, MobileLogCallback {
    func onLog(_ level: String, message: String) {
        print("[\(level)] \(message)")
    }
}

struct Peer {
    let address: String
    let protocol: String
    let region: String
    let rtt: Int64

    init?(from dict: [String: Any]) {
        guard let address = dict["Address"] as? String,
              let protocol = dict["Protocol"] as? String,
              let region = dict["Region"] as? String else {
            return nil
        }

        self.address = address
        self.protocol = protocol
        self.region = region
        self.rtt = dict["RTT"] as? Int64 ?? 0
    }
}
```

### In ViewController

```swift
class PeersViewController: UIViewController {
    private let peerManager = YggdrasilPeerManager()
    private var peers: [Peer] = []

    override func viewDidLoad() {
        super.viewDidLoad()

        Task {
            do {
                peers = try await peerManager.getBestPeers()
                tableView.reloadData()
            } catch {
                print("Failed to load peers: \(error)")
            }
        }
    }
}
```

## Building the AAR

### Windows

```bash
cd mobile
build-android.bat
```

### Linux/macOS

```bash
cd mobile
gomobile bind -target=android -androidapi 23 \
  -ldflags="-checklinkname=0" \
  -o yggpeers.aar \
  github.com/jbselfcompany/yggpeers/mobile
```

## Adding to Android Project

### build.gradle

```gradle
dependencies {
    implementation files('libs/yggpeers.aar')
}
```

### ProGuard Rules

```proguard
-keep class yggpeers.** { *; }
-keep interface yggpeers.** { *; }
```
