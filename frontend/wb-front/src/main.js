import './style.css'

// Конфигурация API
const API_BASE_URL = 'http://localhost:8000/api/v1';

// Элементы DOM
const orderInput = document.getElementById('orderInput');
const searchBtn = document.getElementById('searchBtn');
const loadingElement = document.getElementById('loading');
const errorElement = document.getElementById('error');
const errorMessage = document.getElementById('errorMessage');
const orderResult = document.getElementById('orderResult');

// Обработчики событий
searchBtn.addEventListener('click', handleSearch);
orderInput.addEventListener('keypress', (e) => {
  if (e.key === 'Enter') {
    handleSearch();
  }
});

// Основная функция поиска заказа
async function handleSearch() {
  const orderUid = orderInput.value.trim();

  if (!orderUid) {
    showError('Пожалуйста, введите UID заказа');
    return;
  }

  try {
    showLoading(true);
    hideError();
    hideOrderResult();

    const order = await fetchOrder(orderUid);
    displayOrder(order);
  } catch (error) {
    console.error('Error fetching order:', error);
    showError(error.message || 'Произошла ошибка при поиске заказа');
  } finally {
    showLoading(false);
  }
}

// Запрос к API для получения заказа
async function fetchOrder(uid) {
  const response = await fetch(`${API_BASE_URL}/order/${uid}`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
    },
  });

  if (!response.ok) {
    if (response.status === 404) {
      throw new Error('Заказ с указанным UID не найден');
    } else if (response.status >= 500) {
      throw new Error('Ошибка сервера. Попробуйте позже');
    } else {
      throw new Error(`Ошибка HTTP: ${response.status}`);
    }
  }

  const data = await response.json();
  return data.data; // API возвращает данные в поле data
}

// Отображение информации о заказе
function displayOrder(order) {
  // Основная информация
  document.getElementById('orderUid').textContent = order.order_uid;
  document.getElementById('trackNumber').textContent = order.track_number || 'Не указан';
  document.getElementById('dateCreated').textContent = formatDate(order.date_created);
  document.getElementById('customerId').textContent = order.customer_id || 'Не указан';
  document.getElementById('deliveryService').textContent = order.delivery_service || 'Не указан';

  // Информация о доставке
  displayDeliveryInfo(order.delivery);

  // Информация о платеже
  displayPaymentInfo(order.payment);

  // Список товаров
  displayItems(order.items);

  showOrderResult();
}

// Отображение информации о доставке
function displayDeliveryInfo(delivery) {
  const deliveryContainer = document.getElementById('deliveryInfo');

  if (!delivery) {
    deliveryContainer.innerHTML = '<p class="no-data">Информация о доставке отсутствует</p>';
    return;
  }

  deliveryContainer.innerHTML = `
    <div class="detail-grid">
      <div class="detail-item">
        <label>Имя получателя:</label>
        <span>${delivery.name || 'Не указано'}</span>
      </div>
      <div class="detail-item">
        <label>Телефон:</label>
        <span>${delivery.phone || 'Не указан'}</span>
      </div>
      <div class="detail-item">
        <label>Email:</label>
        <span>${delivery.email || 'Не указан'}</span>
      </div>
      <div class="detail-item">
        <label>Индекс:</label>
        <span>${delivery.zip || 'Не указан'}</span>
      </div>
      <div class="detail-item">
        <label>Город:</label>
        <span>${delivery.city || 'Не указан'}</span>
      </div>
      <div class="detail-item">
        <label>Регион:</label>
        <span>${delivery.region || 'Не указан'}</span>
      </div>
      <div class="detail-item full-width">
        <label>Адрес:</label>
        <span>${delivery.address || 'Не указан'}</span>
      </div>
    </div>
  `;
}

// Отображение информации о платеже
function displayPaymentInfo(payment) {
  const paymentContainer = document.getElementById('paymentInfo');

  if (!payment) {
    paymentContainer.innerHTML = '<p class="no-data">Информация о платеже отсутствует</p>';
    return;
  }

  paymentContainer.innerHTML = `
    <div class="detail-grid">
      <div class="detail-item">
        <label>ID транзакции:</label>
        <span>${payment.transaction || 'Не указан'}</span>
      </div>
      <div class="detail-item">
        <label>Валюта:</label>
        <span>${payment.currency || 'Не указана'}</span>
      </div>
      <div class="detail-item">
        <label>Провайдер:</label>
        <span>${payment.provider || 'Не указан'}</span>
      </div>
      <div class="detail-item">
        <label>Банк:</label>
        <span>${payment.bank || 'Не указан'}</span>
      </div>
      <div class="detail-item">
        <label>Сумма:</label>
        <span>${formatCurrency(payment.amount)} ${payment.currency || ''}</span>
      </div>
      <div class="detail-item">
        <label>Стоимость доставки:</label>
        <span>${formatCurrency(payment.delivery_cost)} ${payment.currency || ''}</span>
      </div>
      <div class="detail-item">
        <label>Сумма товаров:</label>
        <span>${formatCurrency(payment.goods_total)} ${payment.currency || ''}</span>
      </div>
      <div class="detail-item">
        <label>Дополнительные сборы:</label>
        <span>${formatCurrency(payment.custom_fee)} ${payment.currency || ''}</span>
      </div>
    </div>
  `;
}

// Отображение списка товаров
function displayItems(items) {
  const itemsContainer = document.getElementById('itemsList');

  if (!items || items.length === 0) {
    itemsContainer.innerHTML = '<p class="no-data">Товары не найдены</p>';
    return;
  }

  const itemsHtml = items.map(item => `
    <div class="item-card">
      <div class="item-header">
        <h4>${item.name || 'Без названия'}</h4>
        <span class="item-brand">${item.brand || 'Без бренда'}</span>
      </div>
      <div class="item-details">
        <div class="item-detail">
          <label>Артикул:</label>
          <span>${item.chrt_id || 'Не указан'}</span>
        </div>
        <div class="item-detail">
          <label>Размер:</label>
          <span>${item.size || 'Не указан'}</span>
        </div>
        <div class="item-detail">
          <label>Цена:</label>
          <span>${formatCurrency(item.price)}</span>
        </div>
        <div class="item-detail">
          <label>Скидка:</label>
          <span>${item.sale || 0}%</span>
        </div>
        <div class="item-detail">
          <label>Итого:</label>
          <span class="total-price">${formatCurrency(item.total_price)}</span>
        </div>
        <div class="item-detail">
          <label>RID:</label>
          <span>${item.rid || 'Не указан'}</span>
        </div>
      </div>
    </div>
  `).join('');

  itemsContainer.innerHTML = itemsHtml;
}

// Вспомогательные функции для отображения состояний
function showLoading(show) {
  loadingElement.classList.toggle('hidden', !show);
}

function showError(message) {
  errorMessage.textContent = message;
  errorElement.classList.remove('hidden');
}

function hideError() {
  errorElement.classList.add('hidden');
}

function showOrderResult() {
  orderResult.classList.remove('hidden');
}

function hideOrderResult() {
  orderResult.classList.add('hidden');
}

// Вспомогательные функции форматирования
function formatDate(dateString) {
  if (!dateString) return 'Не указана';

  try {
    const date = new Date(dateString);
    return date.toLocaleString('ru-RU', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit'
    });
  } catch (error) {
    return dateString;
  }
}

function formatCurrency(amount) {
  if (amount === null || amount === undefined) return '0';
  return new Intl.NumberFormat('ru-RU').format(amount);
}
