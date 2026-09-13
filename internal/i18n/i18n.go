package i18n

import (
	"fmt"
	"strings"
)

// Supported language codes.
const (
	LangEN = "en"
	LangBN = "bn"
	LangHI = "hi"
)

// LanguageNames maps language codes to their native display names with flag emojis.
var LanguageNames = map[string]string{
	LangEN: "🇺🇸 English",
	LangBN: "🇧🇩 বাংলা (Bengali)",
	LangHI: "🇮🇳 हिन्दी (Hindi)",
}

// NormalizeLanguage validates and normalizes language codes, falling back to English.
func NormalizeLanguage(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	switch lang {
	case LangBN, "bangla", "bengali":
		return LangBN
	case LangHI, "hindi":
		return LangHI
	case LangEN, "english":
		return LangEN
	default:
		return LangEN
	}
}

// LanguageDisplayName returns the display name for a language code.
func LanguageDisplayName(lang string) string {
	code := NormalizeLanguage(lang)
	if name, ok := LanguageNames[code]; ok {
		return name
	}
	return LanguageNames[LangEN]
}

var translations = map[string]map[string]string{
	LangEN: {
		"btn_report":        "🛡️ Report Fraudster",
		"btn_check":         "🔍 Check Identifier",
		"btn_myreports":     "📋 My Submissions",
		"btn_stats":         "📊 Global Statistics",
		"btn_help":          "❓ Help & FAQ",
		"btn_dev":           "👨‍💻 Developer Info",
		"btn_language":      "🌐 Language",
		"btn_submit_report": "🛡️ Submit Fraud Report",
		"btn_check_account": "🔍 Check In Blacklist",
		"btn_cancel":        "❌ Cancel",
		"btn_submit":        "🚀 Submit Report",
		"btn_skip_proof":    "⏩ Skip Proof & Submit",
		"btn_refresh":       "🔄 Refresh",
		"btn_close":         "❌ Close",
		"btn_join_channel":  "📢 Join Channel",
		"btn_i_have_joined": "✅ I Have Joined",
		"btn_back":          "🔙 Back",

		"trigger_scam_detected":    "<b>⚠️ Fraud / Scammer Activity Alert</b>\n\nDid you encounter a fraudster or get scammed? You can submit an official scammer report to <b>@telefraudbot</b> with evidence.\n\nTap the button below to submit a verified report.",
		"trigger_scam_with_target": "<b>⚠️ Fraud / Scammer Activity Alert</b>\n\nDid <b>%s</b> scam you? Tap below to report this user to <b>@telefraudbot</b> with chat/transaction proofs.",

		"start_welcome": "<b>🛡️ Welcome to @telefraudbot</b>\n\nYour community defense network against Telegram fraud.\n\n<b>Developer:</b> %s (%s)\nUse the menu below to report scammers, check identifiers, or view statistics.",
		"dev_info":      "<b>🤖 @telefraudbot System Info</b>\n\n<b>Lead Developer:</b> %s (<code>%s</code>)\n<b>Uptime:</b> %s\n\n<i>Designed & Built for Telegram Group Safety.</i>",
		"help_text":     "<b>❓ Help & FAQ - @telefraudbot</b>\n\n<b>🔍 How to Check an Account:</b>\nSend <code>/check &lt;@username | User ID | Phone&gt;</code> or tap <b>🔍 Check Identifier</b>.\n\n<b>🛡️ How to Report a Fraudster:</b>\nTap <b>🛡️ Report Fraudster</b> to launch the multi-step report wizard, attach screenshots/proof, and submit for admin review.\nQuick submission: <code>/report &lt;target&gt; &lt;reason&gt;</code>\n\n<b>👥 Protecting Your Group:</b>\nAdd @telefraudbot to your group and promote to Administrator. Scammers will be banned automatically upon joining.",

		"lang_select_title":  "<b>🌐 Select Language / ভাষা নির্বাচন / भाषा चयन</b>\n\nPlease select your preferred language below:",
		"lang_updated_user":  "<b>✅ Language Updated!</b>\nYour interface language is now set to <b>%s</b>.",
		"lang_updated_group": "<b>✅ Group Language Updated!</b>\nGroup protection notices will now be sent in <b>%s</b>.",

		"group_settings_title": "<b>⚙️ @telefraudbot Group Protection Dashboard</b>\n\n<b>Group:</b> %s\n<b>Chat ID:</b> <code>%d</code>\n<b>Language:</b> %s\n<b>Last Audited:</b> %s\n\n<i>Tap buttons below to toggle real-time protection features:</i>",
		"group_activated":      "<b>🛡️ @telefraudbot Activated!</b>\n\nThis group is now protected by <b>@telefraudbot</b> — Real-time Telegram Fraud & Scammer Defense Network.\n\n<b>Lead Developer:</b> %s (<code>%s</code>)\n<b>Group Language:</b> %s\n\n<b>Enabled Features:</b>\n• ⚡ <b>Instant Join Gate:</b> Blacklisted fraudsters auto-banned on entry.\n• ⏰ <b>Daily Security Audits:</b> Scheduled background member verification.\n• 🗑️ <b>Scam Message Interception:</b> Fraud handles and scam keyword detection.\n\n<b>Group Admin Commands:</b>\n• <code>/scangroup</code> — Audit current members against blacklist\n• <code>/checkuser &lt;@username | reply&gt;</code> — Verify a member\n• <code>/setlang</code> — Change group language\n• <code>/telefraud_settings</code> — Toggle group protection settings",

		"group_fraud_action":  "<b>⚠️ Blacklisted Fraudster %s!</b>\n\n<b>User:</b> %s (ID: <code>%d</code>)\n<b>Threat Level:</b> %s\n<b>Reason:</b> %s\n\n<i>Protected automatically by @telefraudbot.</i>",
		"group_fraud_mention": "<b>⚠️ Scammer Reference Detected</b>\n\nIdentifier <code>%s</code> mentioned in this chat is listed in the TelefFraud database.\n<b>Threat Level:</b> %s\n<b>Reason:</b> %s",

		"check_prompt":       "<b>🔍 Check Identifier</b>\n\nSend an identifier to look up:\n• Username: <code>@handle</code>\n• User ID: <code>123456789</code>\n• Phone: <code>+1234567890</code>\n\n<i>Or send /cancel to exit.</i>",
		"check_clean":        "<b>✅ No verified fraud record found.</b>\n\nThe identifier is not currently in our active blacklist.",
		"check_found_header": "<b>⚠️ VERIFIED FRAUD RECORD FOUND</b>",
		"check_invalid":      "<b>⚠️ Unrecognized identifier.</b>\n\nSend a username (<code>@handle</code>), user ID, or phone number (<code>+…</code>).",

		"wiz_step1":      "<b>🛡️ Fraudster Report Wizard (Step 1/4)</b>\n\nPlease enter the scammer's identifier:\n• Telegram Username (e.g. <code>@scam_handle</code>)\n• Numeric User ID (e.g. <code>123456789</code>)\n• Phone Number (e.g. <code>+1234567890</code>)\n\n<i>Or tap Cancel below to exit.</i>",
		"wiz_step2":      "<b>🛡️ Fraudster Report Wizard (Step 2/4)</b>\n\n<b>Target:</b> <code>%s</code>\n\nSelect the category that best describes the fraudulent activity:",
		"wiz_step3":      "<b>🛡️ Fraudster Report Wizard (Step 3/4)</b>\n\n<b>Category:</b> %s\n\nPlease write a detailed <b>description</b> of the scam event (what happened, promised goods/services, transaction hash, or fake channels):",
		"wiz_step4":          "<b>🛡️ Fraudster Report Wizard (Step 4/4)</b>\n\nPlease upload <b>proof screenshots, documents, or video recordings</b> of the chat/transaction.\n\n• <b>Proof is mandatory</b> to submit a report.\n• You can send multiple photos or documents.\n• Tap <b>🚀 Submit Report</b> once you have uploaded your proof files.",
		"wiz_submitted":      "<b>✅ Report Submitted Successfully!</b>\n\n<b>Report ID:</b> <code>#%d</code>\n<b>Target:</b> <code>%s</code>\n<b>Category:</b> %s\n<b>Proofs Attached:</b> <code>%d file(s)</code>\n\n<i>Our administration team will review your evidence shortly. You can track progress in <b>📋 My Submissions</b>.</i>",
		"wiz_cancelled":      "<b>❌ Action cancelled.</b> Main menu active.",
		"wiz_short_desc":     "<b>⚠️ Description too short.</b> Please provide more details regarding the incident.",
		"wiz_proof_required": "<b>⚠️ Proof is Mandatory:</b> Please upload at least one screenshot or document proof before submitting.",

		"myreports_empty":  "<b>📋 My Submissions</b>\n\nYou have not submitted any fraud reports yet.\n\nUse <b>🛡️ Report Fraudster</b> or <code>/report &lt;target&gt; &lt;reason&gt;</code> to submit a report.",
		"myreports_header": "<b>📋 My Submitted Reports</b>\n\n",

		"channel_required": "<b>⚠️ Channel Subscription Required</b>\n\nTo access <b>@telefraudbot</b>, you must first join our official updates and verification channel:\n\n👉 <b><a href=\"%s\">Join @telefraud_info Channel</a></b>\n\nAfter joining, tap <b>✅ I Have Joined</b> below to continue.",
	},
	LangBN: {
		"btn_report":        "🛡️ প্রতারক রিপোর্ট করুন",
		"btn_check":         "🔍 আইডি পরীক্ষা করুন",
		"btn_myreports":     "📋 আমার রিপোর্টসমূহ",
		"btn_stats":         "📊 সামগ্রিক পরিসংখ্যান",
		"btn_help":          "❓ সাহায্য ও নিয়মাবলী",
		"btn_dev":           "👨‍💻 ডেভেলপার তথ্য",
		"btn_language":      "🌐 ভাষা পরিবর্তন",
		"btn_submit_report": "🛡️ বাটপার রিপোর্ট জমা দিন",
		"btn_check_account": "🔍 ব্ল্যাকলিস্টে চেক করুন",
		"btn_cancel":        "❌ বাতিল করুন",
		"btn_submit":        "🚀 রিপোর্ট জমা দিন",
		"btn_skip_proof":    "⏩ প্রমাণ ছাড়াই জমা দিন",
		"btn_refresh":       "🔄 রিফ্রেশ",
		"btn_close":         "❌ বন্ধ করুন",
		"btn_join_channel":  "📢 চ্যানেলে যোগ দিন",
		"btn_i_have_joined": "✅ আমি জয়েন করেছি",
		"btn_back":          "🔙 পেছনে যান",

		"trigger_scam_detected":    "<b>⚠️ বাটপার / প্রতারক সতর্কতা!</b>\n\nআপনি কি কোনো প্রতারক বা বাটপারের মুখোমুখি হয়েছেন? <b>@telefraudbot</b>-এ প্রমাণসহ একটি অফিসিয়াল রিপোর্ট জমা দিতে পারেন।\n\nনিচের বাটনে চাপ দিয়ে ভেরিফায়েড রিপোর্ট জমা দিন।",
		"trigger_scam_with_target": "<b>⚠️ বাটপার / প্রতারক সতর্কতা!</b>\n\n<b>%s</b> কি আপনার সাথে বাটপারি বা প্রতারণা করেছে? নিচের বাটনে চাপ দিয়ে প্রমাণসহ রিপোর্ট জমা দিন।",

		"start_welcome": "<b>🛡️ @telefraudbot-এ স্বাগতম</b>\n\nটেলিগ্রাম প্রতারণা ও বাটপারি প্রতিরক্ষায় আপনার কমিউনিটি সিকিউরিটি নেটওয়ার্ক।\n\n<b>ডেভেলপার:</b> %s (%s)\nপ্রতারক রিপোর্ট করতে, আইডি পরীক্ষা করতে নিচের মেনু ব্যবহার করুন।",
		"dev_info":      "<b>🤖 @telefraudbot সিস্টেম তথ্য</b>\n\n<b>প্রধান ডেভেলপার:</b> %s (<code>%s</code>)\n<b>আপটাইম:</b> %s\n\n<i>টেলিগ্রাম গ্রুপ নিরাপত্তার জন্য বিশেষভাবে নির্মিত।</i>",
		"help_text":     "<b>❓ সাহায্য ও নির্দেশিকা - @telefraudbot</b>\n\n<b>🔍 অ্যাকাউন্ট চেক করতে:</b>\n<code>/check &lt;@username | User ID | Phone&gt;</code> পাঠান বা <b>🔍 আইডি পরীক্ষা করুন</b> চাপুন।\n\n<b>🛡️ প্রতারক রিপোর্ট করতে:</b>\n<b>🛡️ প্রতারক রিপোর্ট করুন</b> চাপুন এবং প্রমাণ আপলোড করুন।\nদ্রুত রিপোর্ট: <code>/report &lt;target&gt; &lt;reason&gt;</code>\n\n<b>👥 গ্রুপ রক্ষা করতে:</b>\nগ্রুপে @telefraudbot যুক্ত করুন এবং অ্যাডমিন বানান। প্রতারকরা জয়েন করার সাথে সাথে স্বয়ংক্রিয়ভাবে ব্যান হবে।",

		"lang_select_title":  "<b>🌐 ভাষা নির্বাচন করুন / Select Language</b>\n\nঅনুগ্রহ করে আপনার পছন্দের ভাষা বেছে নিন:",
		"lang_updated_user":  "<b>✅ ভাষা পরিবর্তন সম্পন্ন হয়েছে!</b>\nআপনার বট ভাষা এখন <b>%s</b> করা হয়েছে।",
		"lang_updated_group": "<b>✅ গ্রুপ ভাষা পরিবর্তন সম্পন্ন হয়েছে!</b>\nগ্রুপের সকল নোটিফিকেশন এখন <b>%s</b> ভাষায় পাঠানো হবে।",

		"group_settings_title": "<b>⚙️ @telefraudbot গ্রুপ নিরাপত্তা ড্যাশবোর্ড</b>\n\n<b>গ্রুপ:</b> %s\n<b>চ্যাট আইডি:</b> <code>%d</code>\n<b>ভাষা:</b> %s\n<b>সর্বশেষ স্ক্যান:</b> %s\n\n<i>নিচের বাটন চেপে ফিচার অন/অফ করুন:</i>",
		"group_activated":      "<b>🛡️ @telefraudbot সক্রিয় করা হয়েছে!</b>\n\nএই গ্রুপটি এখন <b>@telefraudbot</b> সিকিউরিটি নেটওয়ার্ক দ্বারা সুরক্ষিত।\n\n<b>প্রধান ডেভেলপার:</b> %s (<code>%s</code>)\n<b>গ্রুপের ভাষা:</b> %s\n\n<b>সক্রিয় ফিচারসমূহ:</b>\n• ⚡ <b>ইনস্ট্যান্ট জয়েন গেট:</b> ব্ল্যাকলিস্টেড বাটপাররা জয়েন করলেই সরাসরি ব্যান হবে।\n• ⏰ <b>দৈনিক সিকিউরিটি অডিট:</b> প্রতিদিন স্বয়ংক্রিয় মেম্বার ভেরিফিকেশন।\n• 🗑️ <b>প্রতারণা শব্দ ও মেসেজ ফিল্টার:</b> বাটপারদের মেসেজ ও লিঙ্ক স্ক্যানিং।\n\n<b>গ্রুপ অ্যাডমিন কমান্ড:</b>\n• <code>/scangroup</code> — বর্তমান মেম্বারদের অডিট করুন\n• <code>/checkuser &lt;@username | reply&gt;</code> — মেম্বার যাচাই করুন\n• <code>/setlang</code> — গ্রুপের ভাষা পরিবর্তন করুন\n• <code>/telefraud_settings</code> — সিকিউরিটি সেটিংস কনফিগার করুন",

		"group_fraud_action":  "<b>⚠️ ব্ল্যাকলিস্টেড প্রতারক %s!</b>\n\n<b>ব্যবহারকারী:</b> %s (ID: <code>%d</code>)\n<b>ঝুঁকির মাত্রা:</b> %s\n<b>কারণ:</b> %s\n\n<i>@telefraudbot দ্বারা স্বয়ংক্রিয়ভাবে সুরক্ষিত।</i>",
		"group_fraud_mention": "<b>⚠️ প্রতারক আইডির উল্লেখ পাওয়া গেছে</b>\n\nএই চ্যাটে উল্লেখিত আইডি <code>%s</code> আমাদের প্রতারক ডাটাবেজে রয়েছে।\n<b>ঝুঁকির মাত্রা:</b> %s\n<b>কারণ:</b> %s",

		"check_prompt":       "<b>🔍 আইডি বা ইউজারনেম পরীক্ষা করুন</b>\n\nঅনুসন্ধানের জন্য পাঠান:\n• ইউজারনেম: <code>@handle</code>\n• ইউজার আইডি: <code>123456789</code>\n• ফোন নম্বর: <code>+88017...</code>\n\n<i>বা বাতিল করতে /cancel পাঠান।</i>",
		"check_clean":        "<b>✅ কোনো প্রতারণার রেকর্ড পাওয়া যায়নি।</b>\n\nএই অ্যাকাউন্টটি আমাদের সক্রিয় ব্ল্যাকলিস্টে তালিকাভুক্ত নেই।",
		"check_found_header": "<b>⚠️ ভেরিফায়েড প্রতারক রেকর্ড পাওয়া গেছে</b>",
		"check_invalid":      "<b>⚠️ সঠিক আইডি প্রদান করুন।</b>\n\nইউজারনেম (<code>@handle</code>), ইউজার আইডি অথবা ফোন নম্বর পাঠান।",

		"wiz_step1":      "<b>🛡️ প্রতারক রিপোর্ট উইজার্ড (ধাপ ১/৪)</b>\n\nপ্রতারকের আইডেন্টিফায়ার প্রদান করুন:\n• টেলিগ্রাম ইউজারনেম (যেমন <code>@scam_handle</code>)\n• ইউজার আইডি (যেমন <code>123456789</code>)\n• ফোন নম্বর (যেমন <code>+88017...</code>)\n\n<i>বা বাতিল করতে নিচে চাপুন।</i>",
		"wiz_step2":      "<b>🛡️ প্রতারক রিপোর্ট উইজার্ড (ধাপ ২/৪)</b>\n\n<b>টার্গেট:</b> <code>%s</code>\n\nপ্রতারণার ধরন (Category) নির্বাচন করুন:",
		"wiz_step3":      "<b>🛡️ প্রতারক রিপোর্ট উইজার্ড (ধাপ ৩/৪)</b>\n\n<b>ক্যাটাগরি:</b> %s\n\nঘটনার বিস্তারিত <b>বিবরণ</b> লিখুন (কিভাবে প্রতারণা করেছে, লেনদেন বা ভুয়া গ্রুপ/চ্যানেলের তথ্য):",
		"wiz_step4":          "<b>🛡️ প্রতারক রিপোর্ট উইজার্ড (ধাপ ৪/৪)</b>\n\nচ্যাট বা লেনদেনের <b>স্ক্রিনশট, ডকুমেন্ট বা ভিডিও প্রমাণ</b> আপলোড করুন।\n\n• রিপোর্ট জমা দেওয়ার জন্য <b>প্রমাণ সংযুক্ত করা বাধ্যতামূলক</b>।\n• আপনি একাধিক ছবি বা ফাইল পাঠাতে পারেন।\n• আপলোড সম্পন্ন হলে <b>🚀 রিপোর্ট জমা দিন</b> চাপুন।",
		"wiz_submitted":      "<b>✅ রিপোর্ট সফলভাবে জমা হয়েছে!</b>\n\n<b>রিপোর্ট আইডি:</b> <code>#%d</code>\n<b>টার্গেট:</b> <code>%s</code>\n<b>ক্যাটাগরি:</b> %s\n<b>সংযুক্ত প্রমাণ:</b> <code>%dটি ফাইল</code>\n\n<i>আমাদের অ্যাডমিন টিম প্রমাণ যাচাই করবে। অগ্রগতির জন্য <b>📋 আমার রিপোর্টসমূহ</b> দেখুন।</i>",
		"wiz_cancelled":      "<b>❌ কার্যক্রম বাতিল করা হয়েছে।</b>",
		"wiz_short_desc":     "<b>⚠️ বিবরণটি খুব সংক্ষিপ্ত।</b> অনুগ্রহ করে আরও বিস্তারিত তথ্য প্রদান করুন।",
		"wiz_proof_required": "<b>⚠️ প্রমাণ আবশ্যক:</b> রিপোর্ট জমা দিতে অন্তত একটি স্ক্রিনশট বা ডকুমেন্ট প্রমাণ আপলোড করুন।",

		"myreports_empty":  "<b>📋 আমার রিপোর্টসমূহ</b>\n\nআপনি এখনও কোনো প্রতারক রিপোর্ট জমা দেননি।\n\nরিপোর্ট করতে <b>🛡️ প্রতারক রিপোর্ট করুন</b> চাপুন।",
		"myreports_header": "<b>📋 আমার জমাকৃত রিপোর্টসমূহ</b>\n\n",

		"channel_required": "<b>⚠️ চ্যানেলে সাবস্ক্রিপশন প্রয়োজন</b>\n\n<b>@telefraudbot</b> ব্যবহার করতে প্রথমে আমাদের অফিসিয়াল চ্যানেলে জয়েন করুন:\n\n👉 <b><a href=\"%s\">Join @telefraud_info Channel</a></b>\n\nজয়েন করার পর নিচে <b>✅ আমি জয়েন করেছি</b> চাপুন।",
	},
	LangHI: {
		"btn_report":        "🛡️ धोखेबाज़ रिपोर्ट करें",
		"btn_check":         "🔍 पहचान जांचें",
		"btn_myreports":     "📋 मेरी रिपोर्ट्स",
		"btn_stats":         "📊 वैश्विक आँकड़े",
		"btn_help":          "❓ सहायता और नियम",
		"btn_dev":           "👨‍💻 डेवलपर जानकारी",
		"btn_language":      "🌐 भाषा बदलें",
		"btn_submit_report": "🛡️ फ्रॉड रिपोर्ट दर्ज करें",
		"btn_check_account": "🔍 ब्लैकलिस्ट जांचें",
		"btn_cancel":        "❌ रद्द करें",
		"btn_submit":        "🚀 रिपोर्ट जमा करें",
		"btn_skip_proof":    "⏩ बिना सबूत जमा करें",
		"btn_refresh":       "🔄 ताज़ा करें",
		"btn_close":         "❌ बंद करें",
		"btn_join_channel":  "📢 चैनल से जुड़ें",
		"btn_i_have_joined": "✅ मैं जुड़ चुका हूँ",
		"btn_back":          "🔙 वापस",

		"trigger_scam_detected":    "<b>⚠️ फ्रॉड / धोखेबाज़ गतिविधि अलर्ट!</b>\n\nक्या आपके साथ धोखाधड़ी हुई है या आपने किसी स्कैमर को देखा है? आप सबूतों के साथ <b>@telefraudbot</b> पर आधिकारिक फ्रॉड रिपोर्ट दर्ज कर सकते हैं।\n\nरिपोर्ट दर्ज करने के लिए नीचे दिए गए बटन पर टैप करें।",
		"trigger_scam_with_target": "<b>⚠️ फ्रॉड / धोखेबाज़ गतिविधि अलर्ट!</b>\n\nक्या <b>%s</b> ने आपके साथ धोखाधड़ी की है? सबूतों के साथ रिपोर्ट दर्ज करने के लिए नीचे टैप करें।",

		"start_welcome": "<b>🛡️ @telefraudbot में आपका स्वागत है</b>\n\nटेलीग्राम धोखाधड़ी के खिलाफ आपका सुरक्षा नेटवर्क।\n\n<b>डेवलपर:</b> %s (%s)\nस्कैमर्स की रिपोर्ट करने या आईडी जांचने के लिए नीचे दिए गए मेनू का उपयोग करें।",
		"dev_info":      "<b>🤖 @telefraudbot सिस्टम जानकारी</b>\n\n<b>मुख्य डेवलपर:</b> %s (<code>%s</code>)\n<b>अपटाइम:</b> %s\n\n<i>टेलीग्राम ग्रुप सुरक्षा के लिए विशेष रूप से निर्मित।</i>",
		"help_text":     "<b>❓ सहायता और दिशानिर्देश - @telefraudbot</b>\n\n<b>🔍 आईडी जांचें:</b>\n<code>/check &lt;@username | User ID | Phone&gt;</code> भेजें या <b>🔍 पहचान जांचें</b> दबाएं।\n\n<b>🛡️ फ्रॉड की रिपोर्ट करें:</b>\n<b>🛡️ धोखेबाज़ रिपोर्ट करें</b> पर टैप करें और सबूत अपलोड करें।\nत्वरित रिपोर्ट: <code>/report &lt;target&gt; &lt;reason&gt;</code>\n\n<b>👥 ग्रुप सुरक्षा:</b>\nअपने ग्रुप में @telefraudbot जोड़ें और एडमिन बनाएं। स्कैमर जुड़ते ही अपने आप बैन हो जाएंगे।",

		"lang_select_title":  "<b>🌐 भाषा चुनें / Select Language</b>\n\nकृपया अपनी पसंदीदा भाषा चुनें:",
		"lang_updated_user":  "<b>✅ भाषा अपडेट हो गई!</b>\nआपकी बॉट भाषा अब <b>%s</b> सेट कर दी गई है।",
		"lang_updated_group": "<b>✅ ग्रुप भाषा अपडेट हो गई!</b>\nग्रुप के सभी सुरक्षा संदेश अब <b>%s</b> में भेजे जाएंगे।",

		"group_settings_title": "<b>⚙️ @telefraudbot ग्रुप सुरक्षा डैशबोर्ड</b>\n\n<b>ग्रुप:</b> %s\n<b>चैट आईडी:</b> <code>%d</code>\n<b>भाषा:</b> %s\n<b>अंतिम ऑडिट:</b> %s\n\n<i>सुरक्षा सेटिंग्स चालू/बंद करने के लिए नीचे टैप करें:</i>",
		"group_activated":      "<b>🛡️ @telefraudbot सक्रिय हो गया!</b>\n\nयह ग्रुप अब <b>@telefraudbot</b> नेटवर्क द्वारा सुरक्षित है।\n\n<b>मुख्य डेवलपर:</b> %s (<code>%s</code>)\n<b>ग्रुप भाषा:</b> %s\n\n<b>सक्रिय सुविधाएं:</b>\n• ⚡ <b>त्वरित जॉइन गेट:</b> ब्लैकलिस्ट किए गए स्कैमर्स जुड़ते ही तुरंत बैन होंगे।\n• ⏰ <b>दैनिक सुरक्षा ऑडिट:</b> ग्रुप सदस्यों का स्वचालित दैनिक सत्यापन।\n• 🗑️ <b>संदेश व कीवर्ड जांच:</b> फ्रॉड हैंडल व शब्दों की पहचान।\n\n<b>ग्रुप एडमिन कमांड:</b>\n• <code>/scangroup</code> — सदस्यों का ऑडिट करें\n• <code>/checkuser &lt;@username | reply&gt;</code> — सदस्य की जांच करें\n• <code>/setlang</code> — ग्रुप की भाषा बदलें\n• <code>/telefraud_settings</code> — सुरक्षा सेटिंग्स बदलें",

		"group_fraud_action":  "<b>⚠️ ब्लैकलिस्टेड धोखेबाज़ %s!</b>\n\n<b>यूज़र:</b> %s (ID: <code>%d</code>)\n<b>जोखिम स्तर:</b> %s\n<b>कारण:</b> %s\n\n<i>@telefraudbot द्वारा स्वचालित रूप से सुरक्षित।</i>",
		"group_fraud_mention": "<b>⚠️ फ्रॉड पहचान का उल्लेख मिला</b>\n\nइस चैट में उल्लेखित आईडी <code>%s</code> हमारे फ्रॉड डेटाबेस में दर्ज है।\n<b>जोखिम स्तर:</b> %s\n<b>कारण:</b> %s",

		"check_prompt":       "<b>🔍 पहचान जांचें</b>\n\nजांचने के लिए भेजें:\n• यूज़रनेम: <code>@handle</code>\n• यूज़र आईडी: <code>123456789</code>\n• फ़ोन नंबर: <code>+9198...</code>\n\n<i>रद्द करने के लिए /cancel भेजें।</i>",
		"check_clean":        "<b>✅ कोई फ्रॉड रिकॉर्ड नहीं मिला।</b>\n\nयह पहचान वर्तमान में ब्लैकलिस्ट में नहीं है।",
		"check_found_header": "<b>⚠️ सत्यापित फ्रॉड रिकॉर्ड मिला!</b>",
		"check_invalid":      "<b>⚠️ अमान्य पहचान।</b>\n\nकृपया यूज़रनेम (<code>@handle</code>), यूज़र आईडी या फ़ोन नंबर भेजें।",

		"wiz_step1":          "<b>🛡️ फ्रॉड रिपोर्ट विज़ार्ड (चरण 1/4)</b>\n\nधोखेबाज़ की पहचान दर्ज करें:\n• टेलीग्राम यूज़रनेम (जैसे <code>@scam_handle</code>)\n• यूज़र आईडी (जैसे <code>123456789</code>)\n• फ़ोन नंबर (जैसे <code>+9198...</code>)\n\n<i>या रद्द करने के लिए नीचे टैप करें।</i>",
		"wiz_step2":          "<b>🛡️ फ्रॉड रिपोर्ट विज़ार्ड (चरण 2/4)</b>\n\n<b>लक्ष्य:</b> <code>%s</code>\n\nधोखाधड़ी की श्रेणी (Category) चुनें:",
		"wiz_step3":          "<b>🛡️ फ्रॉड रिपोर्ट विज़ार्ड (चरण 3/4)</b>\n\n<b>श्रेणी:</b> %s\n\nघटना का विस्तृत <b>विवरण</b> लिखें (क्या हुआ, लेन-देन या फर्जी चैनल):",
		"wiz_step4":          "<b>🛡️ फ्रॉड रिपोर्ट विज़ार्ड (चरण 4/4)</b>\n\nचैट या लेन-देन के <b>स्क्रीनशॉट या दस्तावेज़ सबूत</b> अपलोड करें।\n\n• रिपोर्ट जमा करने के लिए <b>सबूत अनिवार्य है</b>।\n• आप एकाधिक तस्वीरें या दस्तावेज़ भेज सकते हैं।\n• अपलोड पूरा होने पर <b>🚀 रिपोर्ट जमा करें</b> दबाएं।",
		"wiz_submitted":      "<b>✅ रिपोर्ट सफलतापूर्वक जमा हो गई!</b>\n\n<b>रिपोर्ट आईडी:</b> <code>#%d</code>\n<b>लक्ष्य:</b> <code>%s</code>\n<b>श्रेणी:</b> %s\n<b>संलग्न सबूत:</b> <code>%d फ़ाइलें</code>\n\n<i>हमारी एडमिन टीम जल्द ही सबूतों की समीक्षा करेगी।</i>",
		"wiz_cancelled":      "<b>❌ कार्रवाई रद्द कर दी गई।</b>",
		"wiz_short_desc":     "<b>⚠️ विवरण बहुत छोटा है।</b> कृपया अधिक विवरण प्रदान करें।",
		"wiz_proof_required": "<b>⚠️ सबूत अनिवार्य है:</b> कृपया रिपोर्ट सबमिट करने से पहले कम से कम एक स्क्रीनशॉट या दस्तावेज़ अपलोड करें।",

		"myreports_empty":  "<b>📋 मेरी रिपोर्ट्स</b>\n\nआपने अभी तक कोई फ्रॉड रिपोर्ट जमा नहीं की है।\n\nरिपोर्ट करने के लिए <b>🛡️ धोखेबाज़ रिपोर्ट करें</b> दबाएं।",
		"myreports_header": "<b>📋 मेरी जमा की गई रिपोर्ट्स</b>\n\n",

		"channel_required": "<b>⚠️ चैनल सदस्यता आवश्यक है</b>\n\n<b>@telefraudbot</b> का उपयोग करने के लिए पहले हमारे आधिकारिक चैनल से जुड़ें:\n\n👉 <b><a href=\"%s\">Join @telefraud_info Channel</a></b>\n\nजुड़ने के बाद, नीचे <b>✅ मैं जुड़ चुका हूँ</b> पर टैप करें।",
	},
}

// Get retrieves a localized string for a given language and key.
func Get(lang, key string) string {
	code := NormalizeLanguage(lang)
	if dict, ok := translations[code]; ok {
		if val, found := dict[key]; found {
			return val
		}
	}
	// Fallback to English
	if dict, ok := translations[LangEN]; ok {
		if val, found := dict[key]; found {
			return val
		}
	}
	return key
}

// Format retrieves a localized format string and interpolates arguments.
func Format(lang, key string, args ...any) string {
	tmpl := Get(lang, key)
	if len(args) == 0 {
		return tmpl
	}
	return fmt.Sprintf(tmpl, args...)
}
