# Git Generator - Công cụ tạo Commit Message thông minh với AI

<div align="center">

```
   ______    _    __      ______                                       __
  / ____/   (_)  / /_    / ____/  ___    ____   ___    _____  ____ _  / /_  ____    _____
 / / __    / /  / __/   / / __   / _ \  / __ \ / _ \  / ___/ / __ `/ / __/ / __ \  / ___/
/ /_/ /   / /  / /_    / /_/ /  /  __/ / / / //  __/ / /    / /_/ / / /_  / /_/ / / /
\____/   /_/   \__/    \____/   \___/ /_/ /_/ \___/ /_/     \__,_/  \__/  \____/ /_/
```

**Tác giả:** Dao Khoi Nguyen - dknguyen2304@gmail.com  
**Phiên bản:** 1.0.0 - AI-Powered Git Commit Message Generator

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Vietnamese](https://img.shields.io/badge/Language-Vietnamese-red.svg)](README_VI.md)

</div>

## 🚀 Giới thiệu

Git Generator là công cụ CLI mạnh mẽ sử dụng AI (Google Gemini) để tự động tạo commit message chất lượng cao dựa trên các thay đổi trong Git repository của bạn. Công cụ phân tích diff và tạo ra các commit message theo chuẩn conventional commits với best practices.

### ✨ Tính năng nổi bật

- 🤖 **AI-Powered**: Sử dụng Google Gemini AI để tạo commit message thông minh
- 🎨 **Giao diện đẹp**: Banner figlet và giao diện terminal tương tác
- 🇻🇳 **Hỗ trợ tiếng Việt**: Giao diện và thông báo hoàn toàn bằng tiếng Việt
- 📝 **Conventional Commits**: Tuân thủ chuẩn conventional commits
- ⚡ **Chế độ tương tác**: Cấu hình thông qua menu thay vì command line
- 🔍 **Dry-run mode**: Xem trước commit message trước khi áp dụng
- 📊 **Phân tích thông minh**: Phát hiện scope tự động, phân tích context
- ✅ **Validation**: Kiểm tra chất lượng commit message theo Git best practices

## 📦 Cài đặt

### Yêu cầu hệ thống
- Go 1.21 hoặc cao hơn
- Git repository
- Google Gemini API key

### Cài đặt từ source

```bash
# Clone repository
git clone https://github.com/your-username/git-generator.git
cd git-generator

# Build ứng dụng
go build -o git-generator ./cmd/git-generator

# (Tùy chọn) Cài đặt global
go install ./cmd/git-generator
```

## 🔧 Cấu hình

### 1. Khởi tạo cấu hình

```bash
./git-generator init
```

### 2. Thiết lập Gemini API Key

```bash
./git-generator config set-api-key YOUR_GEMINI_API_KEY
```

### 3. Xem cấu hình hiện tại

```bash
./git-generator config show
```

## 🎯 Sử dụng

### Chế độ tương tác (Khuyến nghị)

```bash
./git-generator interactive
# hoặc
./git-generator i
```

Chế độ này cung cấp giao diện menu thân thiện để:
- 🌐 Chọn ngôn ngữ (Tiếng Việt/English)
- 📝 Chọn style commit (Conventional/Traditional/Detailed/Minimal)
- 📏 Cấu hình độ dài tối đa
- ⚙️ Thiết lập các tùy chọn bổ sung

### Chế độ command line

```bash
# Tạo commit message cơ bản
./git-generator generate

# Xem trước không commit
./git-generator generate --dry-run

# Tạo nhiều tùy chọn
./git-generator generate --multiple

# Chọn style cụ thể
./git-generator generate --style conventional
```

### Kiểm tra trạng thái

```bash
./git-generator status
```

## 📋 Các lệnh có sẵn

| Lệnh | Mô tả | Aliases |
|------|-------|---------|
| `generate` | Tạo commit message cho staged changes | `gen`, `g` |
| `interactive` | Chế độ tương tác | `i`, `int` |
| `init` | Khởi tạo cấu hình | - |
| `config show` | Hiển thị cấu hình hiện tại | - |
| `config set-api-key` | Thiết lập API key | - |
| `status` | Hiển thị trạng thái repository | - |

## 🎨 Ví dụ Output

### Banner chào mừng
```
   ______    _    __      ______                                       __
  / ____/   (_)  / /_    / ____/  ___    ____   ___    _____  ____ _  / /_  ____    _____
 / / __    / /  / __/   / / __   / _ \  / __ \ / _ \  / ___/ / __ `/ / __/ / __ \  / ___/
/ /_/ /   / /  / /_    / /_/ /  /  __/ / / / //  __/ / /    / /_/ / / /_  / /_/ / / /
\____/   /_/   \__/    \____/   \___/ /_/ /_/ \___/ /_/     \__,_/  \__/  \____/ /_/

============================================================
Author: Dao Khoi Nguyen - dknguyen2304@gmail.com
Version: 1.0.0 - AI-Powered Git Commit Message Generator
============================================================

🚀 Chào mừng bạn đến với Git Generator!
💡 Công cụ tạo commit message thông minh với AI
```

### Commit message được tạo
```
feat(ui): Add interactive terminal interface with Vietnamese support

Implement comprehensive interactive mode with figlet banner, author information,
and Vietnamese localization. The interface provides menu-driven configuration
instead of command-line arguments, improving user experience significantly.

Key features:
- Figlet banner with application branding
- Author information display
- Vietnamese language support for all UI elements
- Interactive option selection via terminal menus
- Color-coded output for better readability

This enhancement addresses user feedback requesting more intuitive interaction
methods and localization for Vietnamese development teams.
```

## ⚙️ Cấu hình nâng cao

### File cấu hình (.git-generator.yaml)

```yaml
gemini:
  api_key: "your-api-key"
  model: "gemini-1.5-flash"
  temperature: 0.7
  max_tokens: 1000

output:
  style: "conventional"
  language: "vi"
  max_subject_length: 50

git:
  max_diff_size: 10000
  include_file_stats: true

validation:
  enabled: true
  enforce_conventional: true
  check_subject_length: true
  check_imperative_mood: true
```

## 🔍 Tính năng nâng cao

### Phát hiện Scope tự động
- Phân tích đường dẫn file để gợi ý scope phù hợp
- Hỗ trợ các pattern phổ biến (ui, api, db, config, test, docs, ci)
- Ưu tiên scope dựa trên số lượng file thay đổi

### Validation Git Best Practices
- ✅ Subject line không quá 50 ký tự
- ✅ Sử dụng imperative mood
- ✅ Viết hoa chữ cái đầu
- ✅ Body wrap ở 72 ký tự
- ✅ Phát hiện breaking changes
- ✅ Kiểm tra atomic commits

### Phân tích Context thông minh
- 📊 Phân tích git history để hiểu context
- 🔄 Phát hiện pattern thay đổi
- 📈 Đánh giá impact của changes
- 🎯 Gợi ý commit type phù hợp

## 🤝 Đóng góp

Chúng tôi hoan nghênh mọi đóng góp! Vui lòng:

1. Fork repository
2. Tạo feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'feat: add amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Tạo Pull Request

## 📄 License

Dự án này được phân phối dưới MIT License. Xem file [LICENSE](LICENSE) để biết thêm chi tiết.

## 🙏 Cảm ơn

- [Google Gemini AI](https://ai.google.dev/) - AI model mạnh mẽ
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [go-figure](https://github.com/common-nighthawk/go-figure) - ASCII art generator
- [promptui](https://github.com/manifoldco/promptui) - Interactive prompts

---

<div align="center">
Được tạo với ❤️ bởi Dao Khoi Nguyen
</div>
