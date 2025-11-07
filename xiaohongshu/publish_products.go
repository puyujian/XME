package xiaohongshu

import (
	"log/slog"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func addProducts(page *rod.Page, productKeywords []string) error {
	keywords := make([]string, 0, len(productKeywords))
	for _, keyword := range productKeywords {
		trimmed := strings.TrimSpace(keyword)
		if trimmed != "" {
			keywords = append(keywords, trimmed)
		}
	}

	if len(keywords) == 0 {
		return nil
	}

	addButton, err := findAddProductButton(page)
	if err != nil {
		return errors.Wrap(err, "未找到添加商品入口")
	}

	if err := addButton.ScrollIntoView(); err != nil {
		logrus.Debugf("滚动到添加商品按钮失败: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	if err := addButton.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return errors.Wrap(err, "点击添加商品按钮失败")
	}
	time.Sleep(500 * time.Millisecond)

	modal, err := page.Timeout(15 * time.Second).Element("div.multi-goods-selector-modal")
	if err != nil {
		return errors.Wrap(err, "打开商品选择弹窗失败")
	}

	if err := waitForProductListLoad(modal); err != nil {
		logrus.Warnf("等待商品列表加载失败: %v", err)
	}

	searchInput, err := modal.Timeout(10 * time.Second).Element("input[placeholder='搜索商品ID 或 商品名称']")
	if err != nil {
		return errors.Wrap(err, "未找到商品搜索输入框")
	}

	for _, keyword := range keywords {
		if err := inputProductSearchKeyword(searchInput, keyword); err != nil {
			return errors.Wrapf(err, "搜索商品失败: %s", keyword)
		}

		card, err := findProductCard(modal, keyword)
		if err != nil {
			return errors.Wrapf(err, "未找到匹配商品: %s", keyword)
		}

		if err := ensureProductSelected(card); err != nil {
			return errors.Wrapf(err, "选择商品失败: %s", keyword)
		}

		logrus.Infof("已选中商品: %s", keyword)
	}

	saveButton, err := modal.ElementR("div.d-modal-footer button", "保存")
	if err != nil {
		return errors.Wrap(err, "未找到商品保存按钮")
	}

	if err := saveButton.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return errors.Wrap(err, "点击保存商品按钮失败")
	}

	if err := waitForModalClose(page); err != nil {
		return err
	}

	return nil
}

func findAddProductButton(page *rod.Page) (*rod.Element, error) {
	selectors := []string{
		"div.multi-good-select-empty-btn button",
		"div.multi-good-select-add-btn button",
	}

	for _, selector := range selectors {
		elem, err := page.Element(selector)
		if err == nil {
			return elem, nil
		}
	}

	return page.ElementR("button", "添加商品")
}

func inputProductSearchKeyword(input *rod.Element, keyword string) error {
	if _, err := input.Eval(`() => {
		this.focus();
		this.value = '';
		this.dispatchEvent(new Event('input', { bubbles: true }));
	}`); err != nil {
		return err
	}

	if _, err := input.Eval(`(value) => {
		this.focus();
		this.value = value;
		this.dispatchEvent(new Event('input', { bubbles: true }));
		this.dispatchEvent(new Event('change', { bubbles: true }));
	}`, keyword); err != nil {
		return err
	}

	time.Sleep(500 * time.Millisecond)
	return nil
}

func findProductCard(modal *rod.Element, keyword string) (*rod.Element, error) {
	deadline := time.Now().Add(10 * time.Second)
	lowerKeyword := strings.ToLower(keyword)

	for time.Now().Before(deadline) {
		cards, err := modal.Elements(".good-card-container")
		if err != nil || len(cards) == 0 {
			time.Sleep(300 * time.Millisecond)
			continue
		}

		for _, card := range cards {
			nameElem, err := card.Element(".sku-name")
			if err != nil {
				continue
			}

			name, err := nameElem.Text()
			if err != nil {
				continue
			}

			if strings.Contains(strings.ToLower(name), lowerKeyword) {
				return card, nil
			}
		}

		time.Sleep(300 * time.Millisecond)
	}

	return nil, errors.Errorf("未找到商品: %s", keyword)
}

func ensureProductSelected(card *rod.Element) error {
	checkboxInput, err := card.Element("input[type='checkbox']")
	if err != nil {
		return errors.Wrap(err, "未找到商品选择框")
	}

	if res, err := checkboxInput.Eval(`() => {
		if (!this) return false;
		const children = this.children;
		if (children && children.length > 0) {
			for (let i = 0; i < children.length; i++) {
				const child = children[i];
				if (child && (child.textContent || child.innerHTML)) {
					return true;
				}
			}
		}
		return false;
	}`); err == nil {
		if !res.Value.Bool() {
			logrus.Warn("检测到空的选择框元素，跳过此商品")
			return nil
		}
	}

	if res, err := checkboxInput.Eval("() => this.checked"); err == nil && res.Value.Bool() {
		return nil
	}

	if err := card.ScrollIntoView(); err != nil {
		logrus.Debugf("滚动商品卡片失败: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	checkboxArea, err := findCheckboxArea(card)
	if err != nil {
		logrus.Warnf("未找到商品选择区域: %v", err)
	}

	strategies := []func() error{
		func() error {
			if checkboxArea != nil {
				return checkboxArea.Timeout(3*time.Second).Click(proto.InputMouseButtonLeft, 1)
			}
			return nil
		},
		func() error {
			_, err := checkboxInput.Eval(`() => this.click()`)
			return err
		},
		func() error {
			indicators, err := card.Elements(".d-checkbox-indicator")
			if err == nil && len(indicators) > 0 {
				visibleIndicator, err := findVisibleElement(indicators)
				if err == nil && visibleIndicator != nil {
					_, err := visibleIndicator.Eval(`() => this.click()`)
					return err
				}
			}
			return errors.New("未找到可见的复选框指示器")
		},
		func() error {
			_, err := checkboxInput.Eval(`() => {
				this.checked = true;
				this.dispatchEvent(new Event('input', { bubbles: true }));
				this.dispatchEvent(new Event('change', { bubbles: true }));
				return true;
			}`)
			return err
		},
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if res, err := checkboxInput.Eval("() => this.checked"); err == nil && res.Value.Bool() {
			return nil
		}

		for _, strategy := range strategies {
			if err := strategy(); err != nil {
				lastErr = err
				logrus.Debugf("尝试切换商品勾选状态失败: %v", err)
				continue
			}

			time.Sleep(200 * time.Millisecond)

			if res, err := checkboxInput.Eval("() => this.checked"); err == nil && res.Value.Bool() {
				return nil
			}
		}

		time.Sleep(300 * time.Millisecond)
	}

	return errors.Wrapf(lastErr, "商品选择失败（已尝试3次）")
}

func findCheckboxArea(card *rod.Element) (*rod.Element, error) {
	selectors := []string{
		".d-checkbox-main",
		".d-checkbox",
		".product-select-area",
	}

	for _, selector := range selectors {
		elem, err := card.Element(selector)
		if err == nil {
			return elem, nil
		}
	}

	return nil, errors.New("未找到复选框选择区域")
}

func findVisibleElement(elems []*rod.Element) (*rod.Element, error) {
	for _, elem := range elems {
		if isElementVisible(elem) {
			return elem, nil
		}
	}
	return nil, errors.New("未找到可见元素")
}

func waitForProductListLoad(modal *rod.Element) error {
	deadline := time.Now().Add(10 * time.Second)

	for time.Now().Before(deadline) {
		cards, err := modal.Elements(".good-card-container")
		if err == nil && len(cards) > 0 {
			for _, card := range cards {
				if isElementVisible(card) {
					return nil
				}
			}
		}

		emptyStates, err := modal.Elements(".goods-list-empty, .goods-list-search-empty")
		if err == nil {
			for _, empty := range emptyStates {
				if isElementVisible(empty) {
					return nil
				}
			}
		}

		time.Sleep(200 * time.Millisecond)
	}

	return errors.New("等待商品列表加载超时")
}

func waitForModalClose(page *rod.Page) error {
	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		has, _, err := page.Has("div.multi-goods-selector-modal")
		if err == nil && !has {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}

	return errors.New("关闭商品选择弹窗超时")
}

func isElementVisible(elem *rod.Element) bool {
	style, err := elem.Attribute("style")
	if err == nil && style != nil {
		styleStr := *style
		if strings.Contains(styleStr, "left: -9999px") ||
			strings.Contains(styleStr, "top: -9999px") ||
			strings.Contains(styleStr, "position: absolute; left: -9999px") ||
			strings.Contains(styleStr, "display: none") ||
			strings.Contains(styleStr, "visibility: hidden") {
			return false
		}
	}

	visible, err := elem.Visible()
	if err != nil {
		slog.Warn("无法获取元素可见性", "error", err)
		return true
	}

	return visible
}
