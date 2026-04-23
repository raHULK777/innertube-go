package innertube

// ClientType represents which YouTube client to impersonate.
type ClientType string

const (
	ClientWeb         ClientType = "WEB"
	ClientWebMusic    ClientType = "WEB_REMIX"
	ClientAndroid     ClientType = "ANDROID"
	ClientAndroidMusic ClientType = "ANDROID_MUSIC"
	ClientIOS         ClientType = "IOS"
	ClientTVEmbedded  ClientType = "TVHTML5_SIMPLY_EMBEDDED_PLAYER"
)

// clientConfig holds the version and metadata for each client type.
type clientConfig struct {
	Name           string
	Version        string
	APIKey         string
	UserAgent      string
	OSName         string
	OSVersion      string
	Platform       string
	OriginalURL    string
}

var clientConfigs = map[ClientType]clientConfig{
	ClientWeb: {
		Name:        "WEB",
		Version:     "2.20231121.08.00",
		APIKey:      "AIzaSyAO_FJ2SlqU8Q4STEHLGCilw_Y9_11qcW8",
		UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		Platform:    "DESKTOP",
		OriginalURL: baseURL,
	},
	ClientWebMusic: {
		Name:        "WEB_REMIX",
		Version:     "1.20231120.03.01",
		APIKey:      "AIzaSyC9XL3ZjWddXya6X74dJoCTL-WEYFDNX30",
		UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		Platform:    "DESKTOP",
		OriginalURL: musicURL,
	},
	ClientAndroid: {
		Name:      "ANDROID",
		Version:   "19.09.37",
		APIKey:    "AIzaSyA8eiZmM1FaDVjRy-df2KTyQ_vz_yYM39w",
		UserAgent: "com.google.android.youtube/19.09.37 (Linux; U; Android 11) gzip",
		OSName:    "Android",
		OSVersion: "11",
		Platform:  "MOBILE",
	},
	ClientAndroidMusic: {
		Name:      "ANDROID_MUSIC",
		Version:   "6.21.52",
		APIKey:    "AIzaSyAOghZGza2MQSZkY_zfZ370N-PUdXEo8AI",
		UserAgent: "com.google.android.apps.youtube.music/6.21.52 (Linux; U; Android 11) gzip",
		OSName:    "Android",
		OSVersion: "11",
		Platform:  "MOBILE",
	},
	ClientIOS: {
		Name:      "IOS",
		Version:   "19.09.3",
		APIKey:    "AIzaSyB-63vPrdThhKuerbB2N_l7Kwwcxj6yUAc",
		UserAgent: "com.google.ios.youtube/19.09.3 (iPhone14,3; U; CPU iOS 15_6 like Mac OS X)",
		OSName:    "iPhone",
		OSVersion: "15.6",
		Platform:  "MOBILE",
	},
	ClientTVEmbedded: {
		Name:        "TVHTML5_SIMPLY_EMBEDDED_PLAYER",
		Version:     "2.0",
		APIKey:      "AIzaSyAO_FJ2SlqU8Q4STEHLGCilw_Y9_11qcW8",
		UserAgent:   "Mozilla/5.0 (SMART-TV; LINUX; Tizen 6.0) AppleWebKit/538.1 (KHTML, like Gecko) Version/6.0 TV Safari/538.1",
		Platform:    "TV",
		OriginalURL: baseURL,
	},
}

// buildContext returns the InnerTube context payload for the given client.
func (c *Client) buildContext(ct ClientType) map[string]interface{} {
	cfg, ok := clientConfigs[ct]
	if !ok {
		cfg = clientConfigs[ClientWeb]
	}

	clientMap := map[string]interface{}{
		"clientName":    cfg.Name,
		"clientVersion": cfg.Version,
		"hl":            c.opts.Lang,
		"gl":            c.opts.Country,
	}
	if cfg.Platform != "" {
		clientMap["platform"] = cfg.Platform
	}
	if cfg.OSName != "" {
		clientMap["osName"] = cfg.OSName
	}
	if cfg.OSVersion != "" {
		clientMap["osVersion"] = cfg.OSVersion
	}
	if cfg.OriginalURL != "" {
		clientMap["originalUrl"] = cfg.OriginalURL
	}

	ctx := map[string]interface{}{
		"client": clientMap,
	}

	// Inject user data if authenticated
	c.mu.RLock()
	creds := c.credentials
	c.mu.RUnlock()

	if creds != nil && creds.VisitorData != "" {
		clientMap["visitorData"] = creds.VisitorData
	}

	if c.opts.SafeSearch {
		ctx["user"] = map[string]interface{}{
			"lockedSafetyMode": true,
		}
	}

	return ctx
}

// apiKeyFor returns the API key for the given client type.
func apiKeyFor(ct ClientType) string {
	cfg, ok := clientConfigs[ct]
	if !ok {
		return clientConfigs[ClientWeb].APIKey
	}
	return cfg.APIKey
}

// userAgentFor returns the User-Agent string for the given client type.
func userAgentFor(ct ClientType) string {
	cfg, ok := clientConfigs[ct]
	if !ok {
		return clientConfigs[ClientWeb].UserAgent
	}
	return cfg.UserAgent
}
